package event_ctrl

import (
	"github.com/zput/zput_net_golang/net/event"
	"github.com/zput/zput_net_golang/net/protocol"
	"github.com/zput/zput_net_golang/net/multiplex"
)

type EventCtrl struct{
	eventPool map[int]*event.Event
	multi multiplex.Multiplex
}

/*
todo: 这个eventPoll 需要进行加锁？ 还有其他的goroutine对它进行了操作？
*/

func New()*EventCtrl{

}

func (this *EventCtrl)AddEvent(event *event.Event){
	this.eventPool[event.GetFd()]=event
	this.multi.AddEvent(event)
}

func (this *EventCtrl)RemoveEvent(event *event.Event){

	_, ok := this.eventPool[event.GetFd()]
	if ok {
		delete(this.eventPool, event.GetFd())
	}
	this.multi.RemoveEvent(event)
}

func (this *EventCtrl)ModifyEvent(event *event.Event){
	_, ok := this.eventPool[event.GetFd()]
	if !ok {
		// not exist
		this.multi.ModifyEvent(event)
	}
	// if it is already exist, don't need to modify epoll etc.
}

func (this *EventCtrl)WaitAndRunHandle(PollTimeMs int){
	this.multi.WaitEvent(this.handlerEventWrap, PollTimeMs)
}

func (this *EventCtrl) handlerEventWrap(fd int, eventType protocol.EventType) {
	//this.eventHandling.Set(true)

	if fd != -1 {
		event, ok := this.eventPool[fd]
		if ok {
			event.HandleEvent(eventType)
		}else{
			this.RemoveEvent(fd)
		}
	}

	//l.eventHandling.Set(false)
}
