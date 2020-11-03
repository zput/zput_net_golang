package connect

import (
	"errors"
	"fmt"
	"github.com/RussellLuo/timingwheel"
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

type OnMessageCallback func(*Connect, *ringbuffer.RingBuffer)
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
	outBuffer *ringbuffer.RingBuffer // write buffer
	inBuffer  *ringbuffer.RingBuffer // read buffer
	messageCallback OnMessageCallback
	connectCloseCallback OnConnectCloseCallback
	writeCompleteCallback OnWriteCompletCallback
	state ConnectState

	fd        int
	peerAddr  string

	idleTime    time.Duration
	activeTime  protocol.Int64
	timingWheel *timingwheel.TimingWheel
	temporaryBuf []byte
}

var ErrConnectionClosed = errors.New("connection closed")

// New 创建 Connection
func New(loop *event_loop.EventLoop, fd int, sa unix.Sockaddr, tw *timingwheel.TimingWheel, idleTime time.Duration,) (*Connect, error) {
	var tcpConnection = Connect{
		loop:loop,
		fd:fd,
		peerAddr:sockAddrToString(sa),
		outBuffer:pool.Get(),
		inBuffer:pool.Get(),
		state:Disconnected,
		idleTime:idleTime,
		timingWheel:tw,
		temporaryBuf:make([]byte, 0xFFFF),
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

	n, err := unix.Read(this.fd, this.temporaryBuf)
	if n == 0 || err != nil {
		if err != unix.EAGAIN {
			// TODO zxc
			log.Errorf("fd[%d] readEvent error[%v]", this.fd, err)
			this.closeEvent()
		}
		return
	}
	if n > 0{
		_, _ = this.inBuffer.Write(this.temporaryBuf[:n])
		this.messageCallback(this, this.inBuffer)
	}
}

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
}

func (this *Connect) Write(data []byte) {
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
		//设置状态
		this.state = Disconnected
		//在event中取消掉loop注册
		//删除fd-event-loop
		this.event.DisableAll()
		this.event.UnRegister()

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
		this.Write(buffer)
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