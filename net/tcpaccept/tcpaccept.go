package tcpaccept

import (
	"errors"
	"github.com/zput/zput_net_golang/net/log"
	"github.com/zput/zput_net_golang/net/event_loop"
	"github.com/zput/zput_net_golang/net/protocol"
	"net"
	"os"

	reuseport "github.com/libp2p/go-reuseport"
	"golang.org/x/sys/unix"
)

// Listener 监听TCP连接
type Accept struct {
	listener                   net.Listener
	aCopyOfTheUnderlyingOsFile *os.File
	loop                       *event_loop.EventLoop
	newConnectCallback         protocol.OnNewConnectCallback
	event                      *event_loop.Event
}

// New 创建Listener
func New(option protocol.NetWorkAndAddressAndOption, loop *event_loop.EventLoop) (*Accept, error) {
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
	var tcpAccept = Accept{
		listener: listener,
		loop:     loop,
	}

	//从listener中得到FD填充到TcpAccept.
	err = tcpAccept.setFd()
	if err != nil {
		return nil, err
	}
	log.Debugf("created listen fd[%d]; in tcp accept", tcpAccept.Fd())
	//新建Tcp Accept event_loop.
	tcpAccept.event = event_loop.NewEvent(loop, tcpAccept.Fd())
	//将这个accept event添加到loop，给多路复用监听。
	err = tcpAccept.loop.AddEvent(tcpAccept.event)
	if err != nil{
		log.Error("creating tcpAccept failure; AddEvent; error[%v]", err)
		return nil, err
	}

	tcpAccept.event.SetReadFunc(tcpAccept.AcceptHandle)

	return &tcpAccept, nil
}

func (this *Accept)Listen()error{
	log.Debugf("enable reading; in tcp accept activity; FD(%d)", this.event.GetFd())
	return this.event.EnableReading(true)
}

// Close Accept
func (this *Accept) Close()error{
	this.loop.RunInLoop(func() {
		var err error
		err = this.event.DisableAll()
		if err != nil{
			log.Errorf("close event_loop.DisableAll; error[%v]", err)
		}
		err = this.event.UnRegister()
		if err != nil{
			log.Errorf("close event_loop.RemoveFromLoop; error[%v]", err)
		}
		if err := this.listener.Close(); err != nil {
			log.Errorf("[Listener] close; error[%v] ", err)
		}
	})
	return nil
}

func (this *Accept) setFd() error {
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

func (this *Accept) SetNewConnectCallback(newConnectCallback protocol.OnNewConnectCallback) {
	this.newConnectCallback = newConnectCallback
}

func (this *Accept) SetNonblock() error {
	var err error
	//设置非阻塞
	if err = unix.SetNonblock(int(this.aCopyOfTheUnderlyingOsFile.Fd()), true); err != nil {
		return err
	}
	return nil
}

//AcceptHandle供event loop回调处理
func (this *Accept) AcceptHandle() {
	nfd, sa, err := unix.Accept(this.Fd())
	if err != nil {
		if err != unix.EAGAIN {
			log.Error("accept:", err)
		}
		return
	}

	this.newConnectCallback(nfd, sa)
}

// Fd Accept fd
func (this *Accept) Fd() int {
	return int(this.aCopyOfTheUnderlyingOsFile.Fd())
}
