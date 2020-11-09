package event_loop

import (
	"github.com/zput/zput_net_golang/net/log"
	"github.com/zput/zput_net_golang/net/protocol"
	"sync"
)

type EventLoop struct{
	SequenceID int
	eventCtrl * EventCtrl

	// TODO wait add, remove, delete.
	functions []protocol.AddFunToLoopWaitingRun
	mutex sync.Mutex

	running  protocol.Bool
	waitDone chan struct{}
}

func New(sequenceID int)(*EventLoop, error){
	var loop = EventLoop{
		SequenceID: sequenceID,
		functions:make([]protocol.AddFunToLoopWaitingRun, 0),
		waitDone:make(chan struct{}),
	}
	var err error
	// 不-让eventCtrl反向关联这个
	loop.eventCtrl, err = NewEventCtrl()
	if err != nil{
		log.Errorf("create eventCtrl error[%v]; in EventLoop", err)
		return nil, err
	}
	return &loop, nil
}

func (this *EventLoop) Run(){
	this.running.Set(true)

	for {
		this.eventCtrl.waitAndRunHandle(protocol.PollTimeMs)
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

func (this *EventLoop)addEvent(event *Event)error{
	return this.eventCtrl.addEvent(event)
}

func (this *EventLoop)removeEvent(event *Event)error{
	return this.eventCtrl.removeEvent(event)
}

func (this *EventLoop)modifyEvent(event *Event)error{
	return this.eventCtrl.modifyEvent(event)
}

//func(this *EventLoop)AddFunInLoop(fun protocol.AddFunToLoopWaitingRun){
//	this.mutex.Lock()
//	defer this.mutex.Unlock()
//
//	this.functions = append(this.functions, fun)
//}

func (this *EventLoop)runAllFunctionInLoop(){
	// TODO ?这里有race，来自于TcpAccept
	// 难道是因为只要谁得到这个loop就可以添加带运行的闭包函数放入其中，所以有race。

	var functionsTemp []protocol.AddFunToLoopWaitingRun
	{
		this.mutex.Lock()
		functionsTemp = this.functions
		this.functions = nil
		this.mutex.Unlock()
	}
	for i:=0;i<len(functionsTemp);i++{
		functionsTemp[i]()
	}
}

func(this *EventLoop)RunInLoop(fun protocol.AddFunToLoopWaitingRun){
	this.mutex.Lock()
	defer this.mutex.Unlock()

	this.functions = append(this.functions, fun)
	_ = this.wake()
	return
}

func (this *EventLoop)wake()error{
	return this.eventCtrl.wake()
}
