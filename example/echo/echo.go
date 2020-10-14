package main

import (
	"flag"
	"github.com/Allenxuxu/gev/log"
	"github.com/Allenxuxu/ringbuffer"
	"github.com/Allenxuxu/toolkit/sync/atomic"
	"github.com/zput/zput_net_golang/net/event_loop"
	"github.com/zput/zput_net_golang/net/protocol"
	"github.com/zput/zput_net_golang/net/tcpconnect"
	"github.com/zput/zput_net_golang/net/tcpserver"
	_ "net/http/pprof"
	"strconv"
)

//TODO
//_ "net/http/pprof"


type Echo struct {
	tcpserver.HandleEventImpl
	Count atomic.Int64
}

func (this *Echo) ConnectCallback(c *tcpconnect.TcpConnect) {
	this.Count.Add(1)
	this.HandleEventImpl.ConnectCallback(c)
}
func (this *Echo) MessageCallback(c *tcpconnect.TcpConnect, buffer *ringbuffer.RingBuffer) {
	tempData := buffer.Bytes()
	buffer.RetrieveAll()

	c.Write(tempData)
}

func (this *Echo) OnClose(c *tcpconnect.TcpConnect) {
	this.Count.Add(-1)
	this.HandleEventImpl.ConnectCloseCallback(c)
}

func main() {

	handler := new(Echo)
	var port int
	var loops int

	flag.IntVar(&port, "port", 58810, "server port")
	flag.IntVar(&loops, "loops", -1, "num loops")
	flag.Parse()

	mainLoopPtr, err := event_loop.New()
	if err != nil{
		log.Error(err)
	}

	s, err := tcpserver.New(handler, mainLoopPtr,
		protocol.Network("tcp"),
		protocol.Address(":"+strconv.Itoa(port)),
		protocol.NumLoops(loops))
	if err != nil {
		panic(err)
	}

	//s.RunEvery(time.Second*2, func() {
	//	log.Info("connections :", handler.Count.Get())
	//})

	s.Start()
}
