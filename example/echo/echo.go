package main

import (
	"flag"
	"github.com/zput/ringbuffer"
	"github.com/zput/zput_net_golang/net/log"
	"github.com/zput/zput_net_golang/net/protocol"
	"github.com/zput/zput_net_golang/net/tcpconnect"
	"github.com/zput/zput_net_golang/net/tcpserver"
	"net/http"
	_ "net/http/pprof"
	"strconv"
	"sync/atomic"
	"time"
)

type Echo struct {
	tcpserver.HandleEventImpl
	connectTimes int64
}

func (this *Echo) GetConnectTimes() int64 {
	return this.connectTimes
}

func (this *Echo) ConnectCallback(c *tcpconnect.TcpConnect) {
	atomic.AddInt64(&this.connectTimes, 1)
	this.HandleEventImpl.ConnectCallback(c)
}
func (this *Echo) MessageCallback(c *tcpconnect.TcpConnect, buffer *ringbuffer.RingBuffer) {
	first, end := buffer.PeekAll()
	buffer.RetrieveAll()
	out := append(first, end...)
	c.Write(out)
}

func (this *Echo) OnClose(c *tcpconnect.TcpConnect) {
	atomic.AddInt64(&this.connectTimes, -1)
	this.HandleEventImpl.ConnectCloseCallback(c)
}

func main() {
	go func() {
		if err := http.ListenAndServe(":6060", nil); err != nil {
			panic(err)
		}
	}()

	handler := new(Echo)
	var port int
	var loops int

	flag.IntVar(&port, "port", 58810, "server port")
	flag.IntVar(&loops, "loops", -1, "num loops")
	flag.Parse()

	log.Info("server begin")

	s, err := tcpserver.New(handler,
		protocol.Network("tcp"),
		protocol.Address(":"+strconv.Itoa(port)),
		protocol.NumLoops(loops))
	if err != nil {
		panic(err)
	}

	log.Info("created tcpserver successful")

	s.RunEvery(time.Second*20, func() {
		log.Info("connections :", handler.connectTimes)
	})

	s.Start()
	log.Info("server end")
}
