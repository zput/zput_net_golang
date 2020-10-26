package tcpserver

import (
	"github.com/zput/ringbuffer"
	"github.com/zput/zput_net_golang/net/tcpconnect"
)

type IHandleEvent interface {
	ConnectCallback(*tcpconnect.TcpConnect)
	MessageCallback(*tcpconnect.TcpConnect, *ringbuffer.RingBuffer)
	WriteCompletCallback(*tcpconnect.TcpConnect)
	ConnectCloseCallback(*tcpconnect.TcpConnect)
}

type HandleEventImpl struct{}

func(this *HandleEventImpl)ConnectCallback(c *tcpconnect.TcpConnect){
	//log.Infof("connect:[%s]", c.PeerAddr())
}

func(this *HandleEventImpl)MessageCallback(c *tcpconnect.TcpConnect, r *ringbuffer.RingBuffer){
	//log.Infof("connect:[%s] send message[%s]", c.PeerAddr(), r.PrintRingBufferInfo())
}

func(this *HandleEventImpl)WriteCompletCallback(c *tcpconnect.TcpConnect){
	//log.Infof("write complete:[%s]", c.PeerAddr())
}

func(this *HandleEventImpl)ConnectCloseCallback(c *tcpconnect.TcpConnect){
	//log.Infof("connect close:[%s]", c.PeerAddr())
}

