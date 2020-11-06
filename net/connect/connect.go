package connect

import (
	"errors"
	"fmt"
	"github.com/RussellLuo/timingwheel"
	"github.com/panjf2000/gnet/pool/bytebuffer"
	"github.com/zput/ringbuffer"
	"github.com/zput/ringbuffer/pool"
	"github.com/zput/zput_net_golang/net/event_loop"
	"github.com/zput/zput_net_golang/net/log"
	"github.com/zput/zput_net_golang/net/protocol"
	"golang.org/x/sys/unix"
	"net"
	"strconv"
	"time"
)

type OnMessageCallback func(*Connect, []byte)[]byte
type OnConnectCloseCallback func(*Connect)
type OnWriteCompletCallback func(*Connect)

type ConnectState int
const(
	Disconnected ConnectState = 1
	Connecting ConnectState = 2
	Connected ConnectState = 3
	Disconnecting ConnectState = 4
)

// Connection TCP 连接
type Connect struct {
	loop                       *event_loop.EventLoop
	event                      *event_loop.Event
	// TODO init
	buf []byte //
	temporaryBuf []byte // don't need init
	outBuffer *ringbuffer.RingBuffer // write buffer
	inBuffer  *ringbuffer.RingBuffer // read buffer
	// TODO initial
	byteBuffer     *bytebuffer.ByteBuffer // bytes buffer for buffering current packet and data in ring-buffer

	messageCallback OnMessageCallback
	connectCloseCallback OnConnectCloseCallback
	writeCompleteCallback OnWriteCompletCallback
	state ConnectState

	fd        int
	peerAddr  string

	idleTime    time.Duration
	activeTime  protocol.Int64
	timingWheel *timingwheel.TimingWheel
	codeImp protocol.ICodec
}

var ErrConnectionClosed = errors.New("connection closed")

// New 创建 Connection
func New(loop *event_loop.EventLoop, fd int, sa unix.Sockaddr, tw *timingwheel.TimingWheel, idleTime time.Duration, codeImp protocol.ICodec) (*Connect, error) {
	var tcpConnection = Connect{
		loop:loop,
		fd:fd,
		peerAddr:sockAddrToString(sa),
		buf:make([]byte, 0xFFFF),
		outBuffer:pool.Get(),
		inBuffer:pool.Get(),
		codeImp: codeImp,
		state:Disconnected,
		idleTime:idleTime,
		timingWheel:tw,
	}

	tcpConnection.outBuffer.RetrieveAll()
	tcpConnection.inBuffer.RetrieveAll()

	var(
		err error
	)

	if tcpConnection.idleTime > 0 {
		_ = tcpConnection.activeTime.Swap(time.Now().Unix())
		tcpConnection.timingWheel.AfterFunc(tcpConnection.idleTime, tcpConnection.closeTimeoutConn())
	}

	//设置不阻塞
	err = tcpConnection.setNoDelay(true)
	if err != nil{
		return nil, err
	}

	//设置Tcp Accept event_loop.
	tcpConnection.event = event_loop.NewEvent(loop, fd)
	////将这个accept event添加到loop，给多路复用监听。
	//err = tcpConnection.loop.AddEvent(tcpConnection.event)
	//if err != nil{
	//	log.Error("creating tcpConnect failure; AddEvent; error[%v]", err)
	//	return nil, err
	//}

	tcpConnection.event.SetReadFunc(tcpConnection.readEvent)
	tcpConnection.event.SetCloseFunc(tcpConnection.closeEvent)
	tcpConnection.event.SetWriteFunc(tcpConnection.writeEvent)
	tcpConnection.event.SetErrorFunc(tcpConnection.errEvent)

	return &tcpConnection, nil
}

//Close关闭连接
func (this *Connect) Close() error {
	if this.state == Disconnected {
		return ErrConnectionClosed
	}

	this.loop.RunInLoop(func() {
		this.closeEvent()
	})
	return nil
}

func (this *Connect) closeTimeoutConn() func() {
	return func() {
		now := time.Now()
		intervals := now.Sub(time.Unix(this.activeTime.Get(), 0))
		if intervals >= this.idleTime {
			_ = this.Close()
		} else {
			this.timingWheel.AfterFunc(this.idleTime-intervals, this.closeTimeoutConn())
		}
	}
}

func (this *Connect) setNoDelay(enable bool)(err error){
	if err = unix.SetNonblock(this.fd, enable); err != nil {
		_ = unix.Close(this.fd)
		log.Error("set nonblock:", err)
		return
	}
	return nil
}

func (this *Connect) SetMessageCallback(messageCallback OnMessageCallback) {
	this.messageCallback = messageCallback
}

func (this *Connect) SetConnectCloseCallback(connectCloseCallback OnConnectCloseCallback) {
	this.connectCloseCallback = connectCloseCallback
}

func (this *Connect) SetWriteCompleteCallback(writeCompletCallback OnWriteCompletCallback) {
	this.writeCompleteCallback = writeCompletCallback
}

func (this *Connect) ConnectedHandle()(err error){

	//将这个accept event添加到loop，给多路复用监听。
	err = this.event.Register()
	if err != nil{
		log.Error("creating tcpConnect failure; AddEvent; error[%v]", err)
		return err
	}

	this.state = Connected
	err = this.event.EnableReading(true)
	//epoll为电平触发
	/*
	LT 电平触发    高电平触发
	----------------------
		EPOLLIN事件  数据可读
		内核中的socket接收缓冲区 为空  低电平  不会触发
		内核中的socket接收缓冲区 不为空  高电平  会触发
		-----------------------------------------
		EPOLLOUT事件 数据可写
		内核中的socket发送缓冲区不满   高电平
		内核中的socket发送缓冲区 满    低电平

	ET 边沿触发  转换的时候触发
	----------------------
		由低电平-》高电平  才会 触发
		高电平-》低电平 触发
	*/

	//event->enableWriting(true);
	err = this.event.EnableErrorEvent(true)
	return
}

func (this *Connect) readEvent() {
	this.updateActivityTime()

	if !this.outBuffer.IsEmpty() {
		log.Infof("[%d-%d]outBuffer is not empty, read event happened; inBuffer[%d], outBuffer[%d]", this.loop.SequenceID, this.event.GetFd(), this.inBuffer.Size(), this.outBuffer.Size())
		return
	}

	n, err := unix.Read(this.fd, this.buf)
	if n == 0 || err != nil {
		if err != unix.EAGAIN {
			// TODO zxc
			log.Errorf("fd[%d] readEvent error[%v]", this.fd, err)
			this.closeEvent()
		}
		return
	}
	if n > 0{
		this.temporaryBuf = this.buf[:n] // will change by shiftN; ReadN; resetBuffer
		for inFrame, _ := this.read(); inFrame != nil; inFrame, _ = this.read() {
			out := this.messageCallback(this, inFrame)
			if out != nil {
				outFrame, _ := this.codeImp.Encode(this, out)
				if len(outFrame)>0{
					this.write(outFrame)
				}
			}
		}

		this.inBuffer.Write(this.temporaryBuf)
	}
	log.Infof("[%d-%d]R:inBuffer[%d], outBuffer[%d]", this.loop.SequenceID, this.event.GetFd(), this.inBuffer.Size(), this.outBuffer.Size())
}

func (this *Connect) read()([]byte, error){
	result, err := this.codeImp.Decode(this)
	if err != nil{
		log.Errorf("decodeAllDataHaveAccepted; error[%v]", err)
		return nil, err
	}

	return result, nil
}


//func (this *Connect) read(fromRemoteData []byte)[]byte{
//	if this.inBuffer.IsEmpty() == false{
//		var tempSlice = make([]byte, this.inBuffer.Size()+len(fromRemoteData))
//		// TODO
//		// var xxxTest []byte  xxxTest is nil?
//		// type slice {
//		//}
//		first, end := this.inBuffer.PeekAll()
//		if len(first) == 0{
//			copy(tempSlice, end)
//		}else{
//			copy(tempSlice, first)
//			copy(tempSlice[len(first):], end)
//		}
//		copy(tempSlice[this.inBuffer.Size():], fromRemoteData)
//		fromRemoteData = tempSlice
//	}
//	return fromRemoteData
//}
//
//func (this *Connect) decodeAllDataHaveAccepted(fromRemoteData []byte)([]byte, error){
//	result, err := this.codeImp.DeCode(this.read(fromRemoteData))
//	if err != nil{
//		log.Errorf("decodeAllDataHaveAccepted; error[%v]", err)
//		return nil, err
//	}
//
//	return result, nil
//}

func (this *Connect) writeEvent() {
	this.updateActivityTime()

	first, end := this.outBuffer.PeekAll()
	n, err := unix.Write(this.fd, first)
	if err != nil {
		if err == unix.EAGAIN {
			return
		}
		this.closeEvent()
		return
	}
	this.outBuffer.Retrieve(n)

	if n == len(first) && len(end) > 0 {
		n, err = unix.Write(this.fd, end)
		if err != nil {
			if err == unix.EAGAIN {
				return
			}
			this.closeEvent()
			return
		}
		this.outBuffer.Retrieve(n)
	}

	if this.outBuffer.Size() == 0 {
		if this.event.IsWriting() == true{
			this.event.EnableWriting(false)
		}

		//回调写完成函数
		if this.writeCompleteCallback != nil{
			this.writeCompleteCallback(this)
		}
	}
	log.Infof("[%d-%d]W:inBuffer[%d], outBuffer[%d]", this.loop.SequenceID, this.event.GetFd(), this.inBuffer.Size(), this.outBuffer.Size())
}

func (this *Connect) write(data []byte) {
	if !this.outBuffer.IsEmpty(){
		_, _ = this.outBuffer.Write(data)
	} else {
		n, err := unix.Write(this.fd, data)
		if err != nil {
			if err == unix.EAGAIN {
				return
			}
			this.closeEvent()
			return
		}
		if n < len(data) {
			_, _ = this.outBuffer.Write(data[n:])

			if this.outBuffer.Size() > 0 {
				this.event.EnableWriting(true)
			}
		}
	}
}

func (this *Connect) errEvent() {
	this.closeEvent()
}

// TODO 为什么C++需要加share_prt
func (this *Connect) closeEvent() {
	if this.state != Disconnected {
		log.Debug("ready to close connection event")
		//设置状态
		this.state = Disconnected
		//在event中取消掉loop注册
		//删除fd-event-loop
		this.event.DisableAll()
		this.event.UnRegister()

		log.Debug("ready to close connection event; callback upper")
		// 这个是上层的责任，应该由上层来删除。这个TcpConnect与loop，fd的联系。
		if this.connectCloseCallback != nil {
			this.connectCloseCallback(this)
		}

		//没有析构函数，自己释放。
		//TODO close 与 shutdown区别。
		if err := unix.Close(this.fd); err != nil {
			log.Error("[close fd]", err)
		}

		pool.Put(this.inBuffer)
		pool.Put(this.outBuffer)
	}
}

// ShutdownWrite 关闭可写端，等待读取完接收缓冲区所有数据
func (this *Connect) ShutdownWrite() error {
	if this.state == Connected{
		this.state = Disconnecting
		return unix.Shutdown(this.fd, unix.SHUT_WR)
	}
	return nil
}

// PeerAddr 获取客户端地址信息
func (this *Connect) PeerAddr() string {
	return this.peerAddr
}

// Send 用来在非 loop 协程发送
func (this *Connect) WriteInSelfLoop(buffer []byte) error {
	if this.state != Connected {
		return ErrConnectionClosed
	}

	this.loop.RunInLoop(func() {
		this.write(buffer)
	})
	return nil
}

func sockAddrToString(sa unix.Sockaddr) string {
	switch sa := (sa).(type) {
	case *unix.SockaddrInet4:
		return net.JoinHostPort(net.IP(sa.Addr[:]).String(), strconv.Itoa(sa.Port))
	case *unix.SockaddrInet6:
		return net.JoinHostPort(net.IP(sa.Addr[:]).String(), strconv.Itoa(sa.Port))
	default:
		return fmt.Sprintf("(unknown - %T)", sa)
	}
}

func (this *Connect)updateActivityTime(){
	if this.idleTime > 0 {
		_ = this.activeTime.Swap(time.Now().Unix())
	}
}

//////////////////////////public API ////////////////////////////////

func (c *Connect) Read() []byte {
	if c.inBuffer.IsEmpty() {
		return c.temporaryBuf
	}

	c.byteBuffer = bytebuffer.Get()

	first, end := c.inBuffer.PeekAll()
	if len(first) == 0{
		_, _ = c.byteBuffer.Write(end)
	}else{
		_, _ = c.byteBuffer.Write(first)
		_, _ = c.byteBuffer.Write(end)
	}
	_, _ = c.byteBuffer.Write(c.temporaryBuf)
	return c.byteBuffer.Bytes()
}

func (c *Connect) ResetBuffer() {
	c.temporaryBuf = c.temporaryBuf[:0]
	c.inBuffer.Reset()
	bytebuffer.Put(c.byteBuffer)
	c.byteBuffer = nil
}

func (c *Connect) ReadN(n int) (size int, buf []byte) {
	inBufferLen := c.inBuffer.Size()
	tempBufferLen := len(c.temporaryBuf)
	if totalLen := inBufferLen + tempBufferLen; totalLen < n || n <= 0 {
		n = totalLen
	}
	size = n
	if c.inBuffer.IsEmpty() {
		buf = c.temporaryBuf[:n]
		return
	}
	head, tail := c.inBuffer.Peek(n)
	c.byteBuffer = bytebuffer.Get()
	_, _ = c.byteBuffer.Write(head)
	_, _ = c.byteBuffer.Write(tail)
	if inBufferLen >= n {
		buf = c.byteBuffer.Bytes()
		return
	}

	restSize := n - inBufferLen
	_, _ = c.byteBuffer.Write(c.temporaryBuf[:restSize])
	buf = c.byteBuffer.Bytes()
	return
}

func (c *Connect) ShiftN(n int) (size int) {
	inBufferLen := c.inBuffer.Size()
	tempBufferLen := len(c.temporaryBuf)
	if inBufferLen+tempBufferLen < n || n <= 0 {
		c.ResetBuffer()
		size = inBufferLen + tempBufferLen
		return
	}
	size = n
	if c.inBuffer.IsEmpty() {
		c.temporaryBuf = c.temporaryBuf[n:]
		return
	}

	bytebuffer.Put(c.byteBuffer)
	c.byteBuffer = nil

	if inBufferLen >= n {
		c.inBuffer.Retrieve(n)
		return
	}
	c.inBuffer.Reset()

	restSize := n - inBufferLen
	c.temporaryBuf = c.temporaryBuf[restSize:]
	return
}

func (c *Connect) BufferLength() int {
	return c.inBuffer.Size() + len(c.temporaryBuf)
}
