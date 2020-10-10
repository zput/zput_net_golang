package event_loop

import (
	"github.com/zput/zput_net_golang/net/event_ctrl"
	"github.com/zput/zput_net_golang/net/event"
	"github.com/zput/zput_net_golang/net/multiplex"
	"github.com/zput/zput_net_golang/net/protocol"
	"sync"
)

type EventLoop struct{
	multiplexPtr * multiplex.Multiplex
	eventCtrl * event_ctrl.EventCtrl

	functions []protocol.DefaultFunction
	mutex sync.Mutex
}

func New()*EventLoop{
	var loop EventLoop
	// 不-让eventCtrl反向关联这个
	loop.eventCtrl = event_ctrl.New()
	return &loop
}

func (this *EventLoop)AddEvent(event *event.Event){
	this.eventCtrl.AddEvent(event)
}

func (this *EventLoop)RemoveEvent(event *event.Event){
	this.eventCtrl.RemoveEvent(event)
}

func (this *EventLoop)ModifyEvent(event *event.Event){
	this.eventCtrl.ModifyEvent(event)
}

func (this *EventLoop) Run(){
	for {
		this.eventCtrl.WaitAndRunHandle(protocol.PollTimeMs)
		this.runAllFunctionInLoop()
	}
}

func (this *EventLoop)runAllFunctionInLoop(){
	this.mutex.Lock()
	defer this.mutex.Unlock()

	for i:=0;i<len(this.functions);i++{
		this.functions[i]()
	}
	this.functions = nil
}
