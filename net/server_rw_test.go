package net

import (
	"bytes"
	"github.com/zput/ringbuffer"
	"github.com/zput/zput_net_golang/net/connect"
	"github.com/zput/zput_net_golang/net/log"
	"github.com/zput/zput_net_golang/net/protocol"
	"github.com/zput/zput_net_golang/net/server"
	"net"
	"testing"
	"time"
)

type exampleRW struct {
	tcpserver.HandleEventImpl
}

func(this *exampleRW)ConnectCallback(c *connect.Connect){
	log.Infof("connect:[%s]", c.PeerAddr())
}

func(this *exampleRW)MessageCallback(c *connect.Connect, buf *ringbuffer.RingBuffer)[]byte{
	log.Infof("read message:[%s]", c.PeerAddr())
	return buf.ReadAll2NewByteSlice()
}

func(this *exampleRW)WriteCompletCallback(c *connect.Connect){
	log.Infof("write complete:[%s]", c.PeerAddr())
}

func(this *exampleRW)ConnectCloseCallback(c *connect.Connect){
	log.Infof("connect close:[%s]", c.PeerAddr())
}

func Test_zput_net_golang(t *testing.T) {
	var(
		echoString = "hello world"
	)
	log.SetLevel(log.LevelDebug)
	handler := new(exampleRW)

	s, err := tcpserver.New(handler,
		protocol.Network("tcp"),
		protocol.Address(":51833"),
		protocol.NumLoops(1),
		protocol.ReusePort(true))
	if err != nil {
		t.Fatal(err)
	}

	go s.Start()

	conn, err := net.DialTimeout("tcp", "127.0.0.1:51833", time.Second*60)
	if err != nil {
		log.Error(err)
		return
	}

	n, err := conn.Write([]byte(echoString))
	if err != nil {
		t.Fatalf("write error[%v]", err)
	}
	if n != len([]byte(echoString)){
		t.Fatalf("write i/o; expect %d, but get %d", len([]byte(echoString)), n)
	}

	buf := make([]byte, 20)
	n, err = conn.Read(buf)
	if err != nil {
		t.Fatalf("read error[%v]", err)
	}

	if n != len([]byte(echoString)){
		t.Fatalf("read i/o; expect %d, but get %d", len([]byte(echoString)), n)
	}
	if bytes.Equal(buf[:n], []byte(echoString)) == false{
		t.Fatalf("read data is not equivalence; expect %s, but get %s", echoString, string(buf[:n]))
	}

	err = conn.Close()
	if err != nil {
		t.Fatalf("Close Error[%v]", err)
	}

	s.Stop()
}
