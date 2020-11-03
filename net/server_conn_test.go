package net

import (
	"github.com/zput/zput_net_golang/net/log"
	"github.com/zput/zput_net_golang/net/protocol"
	"github.com/zput/zput_net_golang/net/tcpconnect"
	"github.com/zput/zput_net_golang/net/tcpserver"
	"io"
	"net"
	"testing"
	"time"
)

type example2 struct {
	tcpserver.HandleEventImpl
}

func(this *example2)ConnectCallback(c *tcpconnect.Connect){
	log.Infof("connect:[%s]", c.PeerAddr())
	if err := c.Close(); err != nil {
		panic(err)
	}
	log.Infof("[%s], will close", c.PeerAddr())
}

func(this *example2)WriteCompletCallback(c *tcpconnect.Connect){
	log.Infof("write complete:[%s]", c.PeerAddr())
}

func(this *example2)ConnectCloseCallback(c *tcpconnect.Connect){
	log.Infof("connect close:[%s]", c.PeerAddr())
}

func TestConnClose(t *testing.T) {
	log.SetLevel(log.LevelDebug)
	handler := new(example2)

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
	log.Info("---")

	buf := make([]byte, 10)
	n, err := conn.Read(buf)
	if n != 0 || err != io.EOF {
		t.Fatal()
	}
	log.Info(n, "---", string(buf))

	s.Stop()
}
