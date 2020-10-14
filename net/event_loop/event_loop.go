package event_loop

import (
	"github.com/zput/zput_net_golang/net/log"
	"github.com/zput/zput_net_golang/net/event"
	"github.com/zput/zput_net_golang/net/event_ctrl"
	"github.com/zput/zput_net_golang/net/protocol"
	"sync"
)

type EventLoop struct{
	eventCtrl * event_ctrl.EventCtrl

	// TODO wait add, remove, delete.
	functions []protocol.AddFunToLoopWaitingRun
	mutex sync.Mutex

	running  protocol.Bool
	waitDone chan struct{}
}

func New()(*EventLoop, error){
	var loop = EventLoop{
		functions:make([]protocol.AddFunToLoopWaitingRun, 0),
		waitDone:make(chan struct{}),
	}
	var err error
	// 不-让eventCtrl反向关联这个
	loop.eventCtrl, err = event_ctrl.New()
	if err != nil{
		log.Errorf("create eventCtrl error[%v]; in EventLoop", err)
		return nil, err
	}
	return &loop, nil
}

func (this *EventLoop) Run(){
	this.running.Set(true)

	for {
		this.eventCtrl.WaitAndRunHandle(protocol.PollTimeMs)
		this.runAllFunctionInLoop()

		//在tcpaccept,tcpconnect关闭后再关闭。
		if !this.running.Get() {
			close(this.waitDone)
			return
		}
	}
}

func (this *EventLoop)Stop()error{
	if !this.running.Get() {
		return protocol.ErrClosed
	}

	this.running.Set(false)

	<-this.waitDone //https://gfw.go101.org/article/channel.html
	return this.eventCtrl.Stop()
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


func (this *EventLoop)runAllFunctionInLoop(){
	// TODO ?这里有race，来自于TcpAccept
	// 难道是因为只要谁得到这个loop就可以添加带运行的闭包函数放入其中，所以有race。
	this.mutex.Lock()
	defer this.mutex.Unlock()

	for i:=0;i<len(this.functions);i++{
		this.functions[i]()
	}
	this.functions = nil
}

func(this *EventLoop)AddFunInLoop(fun protocol.AddFunToLoopWaitingRun){
	this.mutex.Lock()
	defer this.mutex.Unlock()

	this.functions = append(this.functions, fun)
}
