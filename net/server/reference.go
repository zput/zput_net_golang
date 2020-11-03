package tcpserver

import (
	"github.com/zput/ringbuffer"
	"github.com/zput/zput_net_golang/net/connect"
)

type IHandleEvent interface {
	ConnectCallback(*connect.Connect)
	MessageCallback(*connect.Connect, *ringbuffer.RingBuffer)
	WriteCompletCallback(*connect.Connect)
	ConnectCloseCallback(*connect.Connect)
}

type HandleEventImpl struct{}

func(this *HandleEventImpl)ConnectCallback(c *connect.Connect){
	//log.Infof("connect:[%s]", c.PeerAddr())
}

func(this *HandleEventImpl)MessageCallback(c *connect.Connect, r *ringbuffer.RingBuffer){
	//log.Infof("connect:[%s] send message[%s]", c.PeerAddr(), r.PrintRingBufferInfo())
}

func(this *HandleEventImpl)WriteCompletCallback(c *connect.Connect){
	//log.Infof("write complete:[%s]", c.PeerAddr())
}

func(this *HandleEventImpl)ConnectCloseCallback(c *connect.Connect){
	//log.Infof("connect close:[%s]", c.PeerAddr())
}

