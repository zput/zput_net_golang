# zput_net_golang network library

[![LICENSE](https://img.shields.io/badge/LICENSE-MIT-blue)](https://github.com/zput/zput_net_golang/blob/master/LICENSE)
[![Github Actions](https://github.com/zput/zput_net_golang/workflows/CI/badge.svg)](https://github.com/zput/zput_net_golang/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/zput/zput_net_golang)](https://goreportcard.com/report/github.com/zput/zput_net_golang)
[![GoDoc](https://godoc.org/github.com/zput/zput_net_golang?status.svg)](https://godoc.org/github.com/zput/zput_net_golang)

#### [中文](README-ZH.md) | English

zput_net_golang event-driven (Reactor mode) based on high-performance , non-blocking and lightweight networking framework, not use the standard golang language net network package, 
its multiplexing according to different systems using different system functions (epoll (linux system) and kqueue (FreeBSD/DragonFly/Darwin system)),easy to quickly build high-performance server.

## Features

- Freedom to decide whether to lock or not, balancing performance and thread safety
- Automatically expands space when cache is full
- Provides peek at cached content in advance
- Provide explore class functions that simulate reading first, but don't move the actual


## Performance Testing

<details>
  <summary> 📈 test result </summary>

> os platform: Mac 

### test for write and read

```golang

```

</details>


## Example

<details>
  <summary> echo server</summary>

```go
package main

import (
	"flag"
	"github.com/zput/ringbuffer"
	"github.com/zput/zput_net_golang/net/event_loop"
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

func (this *Echo) ConnectCallback(c *tcpconnect.Connect) {
	atomic.AddInt64(&this.connectTimes, 1)
	this.HandleEventImpl.ConnectCallback(c)
}
func (this *Echo) MessageCallback(c *tcpconnect.Connect, buffer *ringbuffer.RingBuffer) {
	first, end := buffer.PeekAll()
	buffer.RetrieveAll()
	out := append(first, end...)
	c.Write(out)
}

func (this *Echo) OnClose(c *tcpconnect.Connect) {
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
	mainLoopPtr, err := event_loop.New()
	if err != nil {
		log.Error(err)
	}
	log.Info("created event_loop successful")

	s, err := tcpserver.New(handler, mainLoopPtr,
		protocol.Network("tcp"),
		protocol.Address(":"+strconv.Itoa(port)),
		protocol.NumLoops(loops))
	if err != nil {
		panic(err)
	}

	log.Info("created tcpserver successful")

	s.RunEvery(time.Second*2, func() {
		log.Info("connections :", handler.connectTimes)
	})

	s.Start()
	log.Info("server end")
}
```

</details>


## Reference

- [evio](https://github.com/tidwall/evio)
- [muduo](https://github.com/chenshuo/muduo)


## Appendix

welcome pr
