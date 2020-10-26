// +build darwin netbsd freebsd openbsd dragonfly

package multiplex

import (
	"errors"
	"github.com/zput/zput_net_golang/net/event"
	"github.com/zput/zput_net_golang/net/log"
	"github.com/zput/zput_net_golang/net/protocol"
	"golang.org/x/sys/unix"
	"sync"
)

// Multiplex Kqueue封装
type Multiplex struct {
	fd         int // Kqueue fd
	waitEvents []unix.Kevent_t
	sockets    sync.Map // [fd]protocol.EventType
}

func New() (*Multiplex, error) {
	fd, err := unix.Kqueue()
	if err != nil {
		return nil, err
	}
	_, err = unix.Kevent(fd, []unix.Kevent_t{{
		Ident:  0,
		Filter: unix.EVFILT_USER,
		Flags:  unix.EV_ADD | unix.EV_CLEAR,
	}}, nil, nil)
	if err != nil {
		return nil, err
	}

	return &Multiplex{
		fd:         fd,
		waitEvents: make([]unix.Kevent_t, protocol.WaitEventsNumber),
	}, nil
}

//关闭kqueue
func (this *Multiplex) Close() (err error) {
	// TODO error
	_ = unix.Close(this.fd)
	return
}

func (this *Multiplex) AddEvent(ioEvent *event.Event) error {
	log.Debugf("AddEvent; ioEvent; fd:%v, eventType:%v", ioEvent.GetFd(), ioEvent.GetEvents())

	this.sockets.Store(ioEvent.GetFd(), ioEvent.GetEvents())
	kEvents := this.kEvents(protocol.EventNone, ioEvent.GetEvents(), ioEvent.GetFd())
	_, err := unix.Kevent(this.fd, kEvents, nil, nil)
	return err
}

func (this *Multiplex) RemoveEvent(ioEvent *event.Event) error {
	return this.RemoveEventFd(ioEvent.GetFd())
}

func (this *Multiplex) RemoveEventFd(fd int) error {
	v, ok := this.sockets.Load(fd)
	if !ok {
		return errors.New("sync map load error")
	}

	kEvents := this.kEvents(v.(protocol.EventType), protocol.EventNone, fd)
	_, err := unix.Kevent(this.fd, kEvents, nil, nil)
	if err != nil {
		this.sockets.Delete(fd)
	}
	return err
}

func (this *Multiplex) ModifyEvent(ioEvent *event.Event) error {
	oldEvents, ok := this.sockets.Load(ioEvent.GetFd())
	if !ok {
		return errors.New("sync map load error")
	}

	kEvents := this.kEvents(oldEvents.(protocol.EventType), ioEvent.GetEvents(), ioEvent.GetFd())
	_, err := unix.Kevent(this.fd, kEvents, nil, nil)
	if err != nil {
		this.sockets.Store(ioEvent.GetFd(), ioEvent.GetEvents())
	}
	return err
}

func (this *Multiplex) kEvents(old protocol.EventType, new protocol.EventType, fd int) (ret []unix.Kevent_t) {
	if new&protocol.EventRead != 0 {
		if old&protocol.EventRead == 0 {
			ret = append(ret, unix.Kevent_t{Ident: uint64(fd), Flags: unix.EV_ADD, Filter: unix.EVFILT_READ})
		}
	} else {
		if old&protocol.EventRead != 0 {
			ret = append(ret, unix.Kevent_t{Ident: uint64(fd), Flags: unix.EV_DELETE, Filter: unix.EVFILT_READ})
		}
	}

	if new&protocol.EventWrite != 0 {
		if old&protocol.EventWrite == 0 {
			ret = append(ret, unix.Kevent_t{Ident: uint64(fd), Flags: unix.EV_ADD, Filter: unix.EVFILT_WRITE})
		}
	} else {
		if old&protocol.EventWrite != 0 {
			ret = append(ret, unix.Kevent_t{Ident: uint64(fd), Flags: unix.EV_DELETE, Filter: unix.EVFILT_WRITE})
		}
	}
	return
}

// Poll 启动 kqueue 循环
func (this *Multiplex) WaitEvent(embedHandler protocol.EmbedHandler2Multiplex, timeMs int) {

	//var timeOut = unix.Timespec{
	//	Sec: int64(timeMs/1000),
	//	Nsec: 0,
	//}

	var wake bool
	//n, err := unix.Kevent(this.fd, nil, this.waitEvents, &timeOut)
	n, err := unix.Kevent(this.fd, nil, this.waitEvents, nil)
	if err != nil && err != unix.EINTR {
		log.Errorf("EpollWait; error[%v]", err)
		return
	}

	//log.Debugf("in wait event; %d happened", n)

	for i := 0; i < n; i++ {
		fd := int(this.waitEvents[i].Ident)
		if fd != 0 {
			var rEvents protocol.EventType
			if (this.waitEvents[i].Flags&unix.EV_ERROR != 0) || (this.waitEvents[i].Flags&unix.EV_EOF != 0) {
				rEvents |= protocol.EventErr
			}
			if this.waitEvents[i].Filter == unix.EVFILT_WRITE {
				rEvents |= protocol.EventWrite
			}
			if this.waitEvents[i].Filter == unix.EVFILT_READ {
				rEvents |= protocol.EventRead
			}

			embedHandler(fd, rEvents)
		} else {
			wake = true
		}
	}

	if wake {
		log.Debug("i'm wake")
		embedHandler(-1, 0)
		wake = false
	}
	if n == len(this.waitEvents) {
		this.waitEvents = make([]unix.Kevent_t, n*2)
	}
}

// Wake 唤醒 kqueue
func (this *Multiplex) Wake() error {
	_, err := unix.Kevent(this.fd, []unix.Kevent_t{{
		Ident:  0,
		Filter: unix.EVFILT_USER,
		Fflags: unix.NOTE_TRIGGER,
	}}, nil, nil)
	return err
}
