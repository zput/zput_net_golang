package event_loop

import (
	"github.com/Allenxuxu/gev/log"
	"github.com/zput/zput_net_golang/net/event"
	"github.com/zput/zput_net_golang/net/event_ctrl"
	"github.com/zput/zput_net_golang/net/protocol"
	"sync"
)

type EventLoop struct{
	eventCtrl * event_ctrl.EventCtrl

	// TODO wait add, remove, delete.
	functions []protocol.DefaultFunction
	mutex sync.Mutex
}

func New()(*EventLoop, error){
	var loop EventLoop
	var err error
	// 不-让eventCtrl反向关联这个
	loop.eventCtrl, err = event_ctrl.New()
	if err != nil{
		log.Errorf("create eventCtrl error[%v]; in EventLoop", err)
		return nil, err
	}
	return &loop, nil
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
