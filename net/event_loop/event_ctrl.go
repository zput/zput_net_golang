package event_loop

import (
	"github.com/zput/zput_net_golang/net/log"
	"github.com/zput/zput_net_golang/net/protocol"
	"github.com/zput/zput_net_golang/net/multiplex"
)

type EventCtrl struct{
	/*
	   todo: 这个eventPoll 需要进行加锁？ 还有其他的goroutine对它进行了操作？
	*/
	eventPool map[int]*Event
	multi *multiplex.Multiplex
}

func NewEventCtrl()(*EventCtrl, error){
	var eventCtrl EventCtrl
	var err error
	eventCtrl.multi, err = multiplex.New()
	if err != nil{
		log.Errorf("create multiplex error[%v]; in eventCtrl", err)
		return nil, err
	}
	eventCtrl.eventPool = make(map[int]*Event)
	return &eventCtrl, nil
}

func (this *EventCtrl)Stop()error{
	return this.multi.Close()
}

func (this *EventCtrl)AddEvent(eventPtr *Event)error{
	this.eventPool[eventPtr.GetFd()]=eventPtr
	return this.multi.AddEvent(eventPtr.GetFd(), eventPtr.GetEvents(), eventPtr.GetOldEvents())
}

func (this *EventCtrl)RemoveEvent(eventPtr *Event)error{
	_, ok := this.eventPool[eventPtr.GetFd()]
	if ok {
		delete(this.eventPool, eventPtr.GetFd())
	}
	return this.multi.RemoveEvent(eventPtr.GetFd(), eventPtr.GetOldEvents())
}

func (this *EventCtrl)ModifyEvent(eventPtr *Event)error{
	return this.multi.ModifyEvent(eventPtr.GetFd(), eventPtr.GetEvents(), eventPtr.GetOldEvents())
}

func (this *EventCtrl)WaitAndRunHandle(PollTimeMs int){
	this.multi.WaitEvent(this.handlerEventWrap, PollTimeMs)
}

func (this *EventCtrl) handlerEventWrap(fd int, eventType protocol.EventType) {
	//this.eventHandling.Set(true)
	if fd != -1 {
		tempEvent := this.eventPool[fd]
		switch {
		case tempEvent == nil:
			//if err := this.RemoveEventFd(fd); err != nil{
			//	log.Errorf("EventCtrl.RemoveEventFd error; error[%v]",err)
			//}
		default:
			tempEvent.HandleEvent(eventType)
		}
	}
	//l.eventHandling.Set(false)
}

func (this *EventCtrl)Wake()error{
	return this.multi.Wake()
}
