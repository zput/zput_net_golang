package tcpserver

import (
	"fmt"
	"github.com/zput/zput_net_golang/net/tcpconnect"
	"github.com/zput/ringbuffer"
)

type IHandleEvent interface {
	ConnectCallback(*tcpconnect.TcpConnect)
	MessageCallback(*tcpconnect.TcpConnect, *ringbuffer.RingBuffer)
	WriteCompletCallback(*tcpconnect.TcpConnect)
	ConnectCloseCallback(*tcpconnect.TcpConnect)
}

type HandleEventImpl struct{}

func(this *HandleEventImpl)ConnectCallback(c *tcpconnect.TcpConnect){
	fmt.Printf("connect:[%s]", c.PeerAddr())
}

func(this *HandleEventImpl)MessageCallback(c *tcpconnect.TcpConnect, r *ringbuffer.RingBuffer){
	fmt.Printf("connect:[%s] send message[%s]", c.PeerAddr(), r.PrintRingBufferInfo())
}

func(this *HandleEventImpl)WriteCompletCallback(c *tcpconnect.TcpConnect){
	fmt.Printf("write complet:[%s]", c.PeerAddr())
}

func(this *HandleEventImpl)ConnectCloseCallback(c *tcpconnect.TcpConnect){
	fmt.Printf("connect close:[%s]", c.PeerAddr())
}



































