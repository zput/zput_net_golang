package tcpconnect

import (
	"errors"
	"fmt"
	"github.com/zput/zput_net_golang/net/log"
	"github.com/Allenxuxu/ringbuffer"
	"github.com/Allenxuxu/ringbuffer/pool"
	"github.com/Allenxuxu/toolkit/sync/atomic"
	"github.com/zput/zput_net_golang/net/event"
	"github.com/zput/zput_net_golang/net/event_loop"
	"golang.org/x/sys/unix"
	"net"
	"strconv"
	"time"
)

type OnMessageCallback func(*TcpConnect, *ringbuffer.RingBuffer)
type OnConnectCloseCallback func(*TcpConnect)
type OnWriteCompletCallback func(*TcpConnect)


type ConnectState int
const(
	Disconnected ConnectState = 1
	Connecting ConnectState = 2
	Connected ConnectState = 3
	Disconnecting ConnectState = 4
)

// Connection TCP 连接
type TcpConnect struct {
	loop                       *event_loop.EventLoop
	event                      *event.Event
	outBuffer *ringbuffer.RingBuffer // write buffer
	inBuffer  *ringbuffer.RingBuffer // read buffer
	messageCallback OnMessageCallback
	connectCloseCallback OnConnectCloseCallback
	writeCompleteCallback OnWriteCompletCallback
	state ConnectState

	fd        int
	peerAddr  string

	idleTime    time.Duration
	activeTime  atomic.Int64
}

var ErrConnectionClosed = errors.New("connection closed")

// New 创建 Connection
func New(loop *event_loop.EventLoop, fd int, sa unix.Sockaddr) (*TcpConnect, error) {
	var tcpConnection = TcpConnect{
		loop: loop,
		fd:          fd,
		peerAddr:    sockAddrToString(sa),
		outBuffer:   pool.Get(),
		inBuffer:    pool.Get(),
		state: Disconnected,
	}

	//设置不阻塞
	err := tcpConnection.setNoDelay(true)
	if err != nil{
		return nil, err
	}

	//设置Tcp Accept event.
	tcpConnection.event = event.New(loop, fd)
	//将这个accept event添加到loop，给多路复用监听。
	tcpConnection.loop.AddEvent(tcpConnection.event)

	tcpConnection.event.SetReadFunc(tcpConnection.readEvent)
	tcpConnection.event.SetCloseFunc(tcpConnection.readEvent)
	tcpConnection.event.SetWriteFunc(tcpConnection.readEvent)
	tcpConnection.event.SetErrorFunc(tcpConnection.readEvent)

	return &tcpConnection, nil
}

//Close关闭连接
func (this *TcpConnect) Close() error {
	if this.state == Disconnected {
		return ErrConnectionClosed
	}

	this.loop.AddFunInLoop(func() {
		this.closeEvent()
	})
	return nil
}

func (this *TcpConnect) setNoDelay(enable bool)(err error){
	if err = unix.SetNonblock(this.fd, enable); err != nil {
		_ = unix.Close(this.fd)
		log.Error("set nonblock:", err)
		return
	}
	return nil
}

func (this *TcpConnect) SetMessageCallback(messageCallback OnMessageCallback) {
	this.messageCallback = messageCallback
}

func (this *TcpConnect) SetConnectCloseCallback(connectCloseCallback OnConnectCloseCallback) {
	this.connectCloseCallback = connectCloseCallback
}

func (this *TcpConnect) SetWriteCompleteCallback(writeCompletCallback OnWriteCompletCallback) {
	this.writeCompleteCallback = writeCompletCallback
}

func (this *TcpConnect) ConnectedHandle() {
	this.state = Connected
	this.event.EnableReading(true)
	//epoll为电平触发
	//event->enableWriting(true);
	this.event.EnableErrorEvent(true)
}

func (this *TcpConnect) readEvent() {
	buf := make([]byte, 0xFFFF)
	n, err := unix.Read(this.fd, buf)
	if n == 0 || err != nil {
		if err != unix.EAGAIN {
			// TODO zxc
			this.closeEvent()
		}
		return
	}
	if n > 0{
		_, _ = this.inBuffer.Write(buf[:n])
		this.messageCallback(this, this.inBuffer)
	}
}

func (this *TcpConnect) writeEvent(fd int) {
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

	if this.outBuffer.Length() == 0 {
		log.Info("[close write]")
		this.event.EnableWriting(false)

		//回调写完成函数
		if this.writeCompleteCallback != nil{
			this.writeCompleteCallback(this)
		}
	}
}

func (this *TcpConnect) Write(data []byte) {
	if this.outBuffer.Length() > 0 {
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
		if n == 0 {
			_, _ = this.outBuffer.Write(data)
		} else if n < len(data) {
			_, _ = this.outBuffer.Write(data[n:])
		}

		if this.outBuffer.Length() > 0 {
			this.event.EnableWriting(true)
		}
	}
}

func (this *TcpConnect) errEvent() {
	this.closeEvent()
}

// TODO 为什么C++需要加share_prt
func (this *TcpConnect) closeEvent() {
	if this.state != Disconnected {
		//设置状态
		this.state = Disconnected
		//在event中取消掉loop注册
		//删除fd-event-loop
		this.event.DisableAll()
		this.event.RemoveFromLoop()

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
func (this *TcpConnect) ShutdownWrite() error {
	if this.state == Connected{
		this.state = Disconnecting
		return unix.Shutdown(this.fd, unix.SHUT_WR)
	}
	return nil
}

// PeerAddr 获取客户端地址信息
func (this *TcpConnect) PeerAddr() string {
	return this.peerAddr
}

// Send 用来在非 loop 协程发送
func (this *TcpConnect) WriteInSelfLoop(buffer []byte) error {
	if this.state != Connected {
		return ErrConnectionClosed
	}

	this.loop.AddFunInLoop(func() {
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
