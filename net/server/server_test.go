package tcpserver

import (
	"testing"
)

func TestServerUnit(t *testing.T){
	testLoop, err := New(new(HandleEventImpl))
	if err != nil{
		t.Fatal(err)
	}

	var loopTest = []struct{
		in string
		expect string
	}{
		{"aaa", "aaa"},
		{"bbb", "bbb"},
	}

	for _, tt := range loopTest{
		testLoop.addConnect(tt.in, nil)
		if _, ok := testLoop.connectPool[tt.in]; !ok{
			t.Fatal("expect exist, get not exist")
		}

		testLoop.removeConnect(tt.in)
		if _, ok := testLoop.connectPool[tt.in]; ok{
			t.Fatal("expect not exist, get exist")
		}
	}
}
