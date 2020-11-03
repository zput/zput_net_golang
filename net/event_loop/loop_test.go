package event_loop

import (
	"github.com/zput/zput_net_golang/net/protocol"
	"testing"
)

func TestEventUpdate(t *testing.T){
	var(
		tempFD= 888
	)
	loop, err := New(1)
	if err != nil {
		t.Fatalf("create fd error; error[%+v]", err)
	}
	var eventTest = NewEvent(loop, tempFD)
	err = eventTest.Register()
	if err != nil {
		t.Fatalf("create fd error; error[%+v]", err)
	}
	if _, ok := loop.eventCtrl.eventPool[tempFD]; !ok{
		t.Fatalf("expect tempFD[%d] is exist, get not exist", tempFD)
	}

	//read
	err = eventTest.EnableReading(true)
	if err != nil{
		t.Fatal(err)
	}
	eventSub(eventTest, protocol.EventRead, t)
	//write
	err = eventTest.EnableWriting(true)
	if err != nil{
		t.Fatal(err)
	}
	eventSub(eventTest, protocol.EventWrite, t)
	// err
	err = eventTest.EnableErrorEvent(true)
	if err != nil{
		t.Fatal(err)
	}
	eventSub(eventTest, protocol.EventErr, t)

	err = eventTest.UnRegister()
	if err != nil {
		t.Fatalf("unregister fd error; error[%+v]", err)
	}
	if _, ok := loop.eventCtrl.eventPool[tempFD]; ok{
		t.Fatalf("expect tempFD[%d] is not exist, get exist", tempFD)
	}
}

func eventSub(eventTest *Event, eventState protocol.EventType, t *testing.T){
	if eventTest.events == eventTest.oldEvents{
		t.Fatal("expect events is not equal oldevent, get result is same")
	}
	if eventTest.events & eventState == protocol.EventNone{
		t.Fatalf("expect events including [%d] event, get non [%d] event", eventState, eventState)
	}
}

