package tcpaccept

import (
	"errors"
	"github.com/zput/zput_net_golang/net/log"
	"github.com/zput/zput_net_golang/net/event"
	"github.com/zput/zput_net_golang/net/event_loop"
	"github.com/zput/zput_net_golang/net/protocol"
	"net"
	"os"

	reuseport "github.com/libp2p/go-reuseport"
	"golang.org/x/sys/unix"
)

// Listener 监听TCP连接
type TcpAccept struct {
	listener                   net.Listener
	aCopyOfTheUnderlyingOsFile *os.File
	loop                       *event_loop.EventLoop
	newConnectCallback         protocol.OnNewConnectCallback
	event                      *event.Event
}

// New 创建Listener
func New(option protocol.NetWorkAndAddressAndOption, loop *event_loop.EventLoop) (*TcpAccept, error) {
	var (
		listener net.Listener
		err error
	)
	if option.ReusePort {
		listener, err = reuseport.Listen(option.Network, option.Address)
	} else {
		listener, err = net.Listen(option.Network, option.Address)
	}
	if err != nil {
		return nil, err
	}
	var tcpAccept = TcpAccept{
		listener: listener,
		loop:     loop,
	}

	//从listener中得到FD填充到TcpAccept.
	err = tcpAccept.setFd()
	if err != nil {
		return nil, err
	}
	log.Debugf("created listen fd[%d]; in tcp accept", tcpAccept.Fd())
	//新建Tcp Accept event.
	tcpAccept.event = event.New(loop, tcpAccept.Fd())
	//将这个accept event添加到loop，给多路复用监听。
	err = tcpAccept.loop.AddEvent(tcpAccept.event)
	if err != nil{
		log.Error("creating tcpAccept failure; AddEvent; error[%v]", err)
		return nil, err
	}

	tcpAccept.event.SetReadFunc(tcpAccept.AcceptHandle)

	return &tcpAccept, nil
}

func (this *TcpAccept)Listen()error{
	log.Debugf("enable reading; in tcp accept activity; FD(%d)", this.event.GetFd())
	return this.event.EnableReading(true)
}

// Close TcpAccept
func (this *TcpAccept) Close()error{
	this.loop.AddFunInLoop(func() {
		var err error
		err = this.event.DisableAll()
		if err != nil{
			log.Errorf("close event.DisableAll; error[%v]", err)
		}
		err = this.event.RemoveFromLoop()
		if err != nil{
			log.Errorf("close event.RemoveFromLoop; error[%v]", err)
		}
		if err := this.listener.Close(); err != nil {
			log.Errorf("[Listener] close; error[%v] ", err)
		}
	})
	return nil
}

func (this *TcpAccept) setFd() error {
	tcpListener, ok := this.listener.(*net.TCPListener)
	if !ok {
		return errors.New("could not get file descriptor")
	}
	file, err := tcpListener.File()
	if err != nil {
		return err
	}
	this.aCopyOfTheUnderlyingOsFile = file
	return nil
}

func (this *TcpAccept) SetNewConnectCallback(newConnectCallback protocol.OnNewConnectCallback) {
	this.newConnectCallback = newConnectCallback
}

func (this *TcpAccept) SetNonblock() error {
	var err error
	//设置非阻塞
	if err = unix.SetNonblock(int(this.aCopyOfTheUnderlyingOsFile.Fd()), true); err != nil {
		return err
	}
	return nil
}

//AcceptHandle供event loop回调处理
func (this *TcpAccept) AcceptHandle() {
	nfd, sa, err := unix.Accept(this.Fd())
	if err != nil {
		if err != unix.EAGAIN {
			log.Error("accept:", err)
		}
		return
	}

	this.newConnectCallback(nfd, sa)
}

// Fd TcpAccept fd
func (this *TcpAccept) Fd() int {
	return int(this.aCopyOfTheUnderlyingOsFile.Fd())
}
