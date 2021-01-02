# zput_net_golang网络库

[![LICENSE](https://img.shields.io/badge/LICENSE-MIT-blue)](https://github.com/zput/zput_net_golang/blob/master/LICENSE)
[![Github Actions](https://github.com/zput/zput_net_golang/workflows/CI/badge.svg)](https://github.com/zput/zput_net_golang/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/zput/zput_net_golang)](https://goreportcard.com/report/github.com/zput/zput_net_golang)
[![GoDoc](https://godoc.org/github.com/zput/zput_net_golang?status.svg)](https://godoc.org/github.com/zput/zput_net_golang)

#### 中文 | [English](README.md)

zput_net_golang基于事件驱动(Reactor模式)的高性能,非阻塞和轻量级网络框架，不使用标准golang语言net网络包, 它的多路复用根据不同系统使用不同的系统函数(epoll(linux系统)和kqueue(FreeBSD系统)), 轻松快速搭建高性能服务器.
    
    
## 特点

- 非阻塞I / O。
- 多Goroutine支持，每个Goroutine运行一个事件驱动的事件循环。
- 读写缓冲区使用可伸缩的环形缓冲区。
- 支持端口重用（SO_REUSEPORT）。
- 支持事件定时任务。

## 性能测试

<details>
  <summary> 📈 测试数据 </summary>

> 测试电脑 Mac 

### 读写测试

```golang

```

</details>

## 安装

```bash
go get -u github.com/zput/ringbuffer
```

## 示例

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
	"github.com/zput/zput_net_golang/net/connect"
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


## 参考

- [evio](https://github.com/tidwall/evio)
- [muduo](https://github.com/chenshuo/muduo)

## 附录

欢迎PR
