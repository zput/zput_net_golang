package event

import (
	"github.com/zput/zput_net_golang/net/event_loop"
	"github.com/zput/zput_net_golang/net/protocol"
)

type Event struct{
	eventFd int
	events protocol.EventType

	eventLoop *event_loop.EventLoop

	readHandle protocol.DefaultFunction
	writeHandle protocol.DefaultFunction
	errorHandle protocol.DefaultFunction
	closeHandle protocol.DefaultFunction
}

func (this *Event)EnableReading(isEnable bool){
	if isEnable{
		this.events |= protocol.EventRead
	}else{
		this.events &= ^protocol.EventRead
	}
	this.update()
}

func (this *Event)EnableWriting(isEnable bool){
	if isEnable{
		this.events |= protocol.EventWrite
	}else{
		this.events &= ^protocol.EventWrite
	}
	this.update()
}

func (this *Event)EnableErrorEvent(isEnable bool){
	if isEnable{
		this.events |= protocol.EventErr
	}else{
		this.events &= ^protocol.EventErr
	}
	this.update()
}

func (this *Event)DisableAll(){
	this.events = protocol.EventNone
	this.update()
}

func (this *Event)IsWriting()bool{
	if this.events & protocol.EventWrite == protocol.EventNone{
		return false
	}
	return true
}

func (this *Event)isReading()bool{
	if this.events & protocol.EventRead == protocol.EventNone{
		return false
	}
	return true
}

func (this *Event)GetFd()int{
	return this.eventFd
}

func (this *Event)GetEvents()protocol.EventType{
	return this.events
}

func(this *Event)SetReadFunc(function protocol.DefaultFunction){
	this.readHandle = function
}

func(this *Event)SetWriteFunc(function protocol.DefaultFunction){
	this.writeHandle = function
}

func(this *Event)SetErrorFunc(function protocol.DefaultFunction){
	this.errorHandle = function
}

func(this *Event)SetCloseFunc(function protocol.DefaultFunction){
	this.closeHandle = function
}

func (this *Event)update(){
	this.eventLoop.ModifyEvent(this)
}

func (this *Event)RemoveFromLoop(){
	this.eventLoop.RemoveEvent(this)
}

func (this *Event)HandleEvent(revents protocol.EventType){
	if (revents & protocol.EventClose) != protocol.EventNone{
		if this.closeHandle != nil{
			this.closeHandle()
		}
	}
	if (revents & protocol.EventErr) != protocol.EventNone{
		if this.errorHandle != nil{
			this.errorHandle()
		}
	}
	if (revents & protocol.EventRead) != protocol.EventNone{
		if this.readHandle != nil {
			this.readHandle()
		}
	}
	if (revents & protocol.EventWrite) != protocol.EventNone{
		if this.writeHandle != nil{
			this.writeHandle()
		}
	}
}
