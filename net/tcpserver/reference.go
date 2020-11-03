package tcpserver

import (
	"github.com/zput/ringbuffer"
	"github.com/zput/zput_net_golang/net/tcpconnect"
)

type IHandleEvent interface {
	ConnectCallback(*tcpconnect.Connect)
	MessageCallback(*tcpconnect.Connect, *ringbuffer.RingBuffer)
	WriteCompletCallback(*tcpconnect.Connect)
	ConnectCloseCallback(*tcpconnect.Connect)
}

type HandleEventImpl struct{}

func(this *HandleEventImpl)ConnectCallback(c *tcpconnect.Connect){
	//log.Infof("connect:[%s]", c.PeerAddr())
}

func(this *HandleEventImpl)MessageCallback(c *tcpconnect.Connect, r *ringbuffer.RingBuffer){
	//log.Infof("connect:[%s] send message[%s]", c.PeerAddr(), r.PrintRingBufferInfo())
}

func(this *HandleEventImpl)WriteCompletCallback(c *tcpconnect.Connect){
	//log.Infof("write complete:[%s]", c.PeerAddr())
}

func(this *HandleEventImpl)ConnectCloseCallback(c *tcpconnect.Connect){
	//log.Infof("connect close:[%s]", c.PeerAddr())
}

