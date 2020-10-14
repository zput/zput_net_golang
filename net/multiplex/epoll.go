// +build linux

package multiplex

import (
	"github.com/zput/zput_net_golang/net/log"
	"github.com/zput/zput_net_golang/net/event"
	"github.com/zput/zput_net_golang/net/protocol"
	"golang.org/x/sys/unix"
)

const readEvent = unix.EPOLLIN | unix.EPOLLPRI
const writeEvent = unix.EPOLLOUT
const errEvent = unix.EPOLLERR
const nonEvent = 0

const waitEventsNumber = 1024

// Multiplex Epoll封装
type Multiplex struct {
	fd       int // epoll fd
	wakeEventFd  int // 用户唤醒的作用file describe
	waitEvents []unix.EpollEvent
}

// 创建Poller对象
func New() (*Multiplex, error) {
	fd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return nil, err
	}

	wakeEventFd, err := newWakeFd(fd)
	if err != nil{
		return nil, err
	}

	return &Multiplex{
		fd:       fd,
		wakeEventFd:  wakeEventFd,
		waitEvents : make([]unix.EpollEvent, waitEventsNumber),
	}, nil
}

//Close关闭epoll
func (this *Multiplex) Close() (err error) {
	//TODO error
	_ = unix.Close(this.fd)
	_ = unix.Close(this.wakeEventFd)
	return
}

func newWakeFd(epollFd int)(int, error){

	r0, _, errno := unix.Syscall(unix.SYS_EVENTFD2, 0, 0, 0)
	if errno != 0 {
		return 0, errno
	}
	wakeEventFd := int(r0)

	err := unix.EpollCtl(epollFd, unix.EPOLL_CTL_ADD, wakeEventFd, &unix.EpollEvent{
		Events: unix.EPOLLIN,
		Fd:     int32(wakeEventFd),
	})
	if err != nil {
		_ = unix.Close(epollFd)
		_ = unix.Close(wakeEventFd)
		return 0, err
	}
	return wakeEventFd, nil
}

func GetEpollEventsFromIOEvent(eventType protocol.EventType)(Events uint32){
	var epollEvents uint32
	epollEvents = nonEvent

	if (eventType & protocol.EventErr) > 0{
		epollEvents |= errEvent
	}

	if (eventType & protocol.EventRead) > 0{
		epollEvents |= readEvent
	}

	if (eventType & protocol.EventWrite) > 0{
		epollEvents |= writeEvent
	}
	return epollEvents
}

func(this *Multiplex)epollCtrl(op int, fd int, eventType protocol.EventType)error{

	var epollEvent = unix.EpollEvent{
		Events : GetEpollEventsFromIOEvent(eventType),
		Fd: int32(fd),
	}
	return unix.EpollCtl(this.fd, op, fd, &epollEvent)
}

func(this *Multiplex) AddEvent(ioEvent *event.Event)error{
	if err := this.epollCtrl(unix.EPOLL_CTL_ADD, ioEvent.GetFd(), ioEvent.GetEvents()); err != nil{
		log.Errorf("add epoll error[%v]", err)
		return err
	}
	return nil
}

func(this *Multiplex) RemoveEvent(ioEvent *event.Event)error{
	if err := this.epollCtrl(unix.EPOLL_CTL_DEL, ioEvent.GetFd(), ioEvent.GetEvents()); err != nil{
		log.Error("remove epoll error[%v]", err)
		return err
	}
	return nil
}

func(this *Multiplex) RemoveEventFd(fd int)error{
	if err := this.epollCtrl(unix.EPOLL_CTL_DEL, fd, protocol.EventNone);err != nil{
		log.Error("remove epoll error[%v]", err)
		return err
	}
	return nil
}

func(this *Multiplex) ModifyEvent(ioEvent *event.Event)error{
	if err := this.epollCtrl(unix.EPOLL_CTL_MOD, ioEvent.GetFd(), ioEvent.GetEvents()); err != nil{
		log.Error("modify epoll error[%v]", err)
		return err
	}
	return nil
}

func(this *Multiplex)WaitEvent(embedHandler protocol.EmbedHandler2Multiplex, timeMs int)(){

	var wake bool

	n, err := unix.EpollWait(this.fd, this.waitEvents, timeMs)

	if err != nil && err != unix.EINTR {
		log.Error("EpollWait: ", err)
		return
	}

	for i := 0; i < n; i++ {
		fd := int(this.waitEvents[i].Fd)
		if fd != this.wakeEventFd {
			var rEvents protocol.EventType
			if ((this.waitEvents[i].Events & unix.POLLHUP) != 0) && ((this.waitEvents[i].Events & unix.POLLIN) == 0) {
				rEvents |= protocol.EventClose
			}
			if this.waitEvents[i].Events&unix.EPOLLERR != 0{
				rEvents |= protocol.EventErr
			}
			if this.waitEvents[i].Events&(unix.EPOLLIN|unix.EPOLLPRI|unix.EPOLLRDHUP) != 0 {
				rEvents |= protocol.EventRead
			}
			if this.waitEvents[i].Events&unix.EPOLLOUT != 0 {
				rEvents |= protocol.EventWrite
			}
			embedHandler(fd, rEvents)
		} else {
			this.wakeHandlerRead()
			wake = true
		}

		if wake {
			embedHandler(-1, 0)
			wake = false
		}
	}

	if n == len(this.waitEvents) {
		this.waitEvents = make([]unix.EpollEvent, n*2)
	}
}

//--------------wake up AND receive wake data------------------
var wakeBytes = []byte{1, 0, 0, 0, 0, 0, 0, 0}

// Wake 唤醒 epoll
func (this *Multiplex) Wake() error {
	_, err := unix.Write(this.wakeEventFd, wakeBytes)
	return err
}

var buf = make([]byte, 8)

func (this *Multiplex) wakeHandlerRead() {
	n, err := unix.Read(this.wakeEventFd, buf)
	if err != nil || n != 8 {
		log.Error("wakeHandlerRead", err, n)
	}
}
//---------------------------------------------

