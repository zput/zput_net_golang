//// Copyright 2019 Andy Pan. All rights reserved.
//// Use of this source code is governed by an MIT-style
//// license that can be found in the LICENSE file.
//
//// +build darwin netbsd freebsd openbsd dragonfly linux
//
package tcpserver
//
//import (
//	"github.com/zput/zput_net_golang/net/event_loop"
//	"github.com/zput/zput_net_golang/net/connect"
//	"time"
//
//	"github.com/smartystreets-prototypes/go-disruptor"
//	"github.com/zput/ringbuffer"
//)
//
//const (
//	// connRingBufferSize indicates the size of disruptor ring-buffer.
//	connRingBufferSize = 1024 * 64
//	connRingBufferMask = connRingBufferSize - 1
//	disruptorCleanup   = time.Millisecond * 10
//
//	cacheRingBufferSize = 1024
//)
//
//type mail struct {
//	fd   int
//	conn *tcpconnect.Connect
//}
//
//type eventConsumer struct {
//	numLoops       int
//	numLoopsMask   int
//	loop           *event_loop.EventLoop
//	connRingBuffer *[connRingBufferSize]*tcpconnect.Connect
//}
//
//func (ec *eventConsumer) Consume(lower, upper int64) {
//	for ; lower <= upper; lower++ {
//		// Connections load balance under round-robin algorithm.
//		if ec.numLoops > 1 {
//			// Leverage "and" operator instead of "modulo" operator to speed up round-robin algorithm.
//			idx := int(lower) & ec.numLoopsMask
//			if idx != ec.loop.SequenceID {
//				// Don't match the round-robin rule, ignore this connection.
//				continue
//			}
//		}
//
//		conn := ec.connRingBuffer[lower&connRingBufferMask]
//		conn.inBuf = ringbuffer.New(cacheRingBufferSize)
//		conn.outBuf = ringbuffer.New(cacheRingBufferSize)
//		conn.loop = ec.loop
//
//		_ = ec.loop.poller.Trigger(&mail{fd: conn.fd, conn: conn})
//	}
//}
//
//func activateMainReactor(svr *Server) {
//	defer func() {
//		time.Sleep(disruptorCleanup)
//	}()
//
//	var connRingBuffer = &[connRingBufferSize]*tcpconnect.Connect{}
//
//	eventConsumers := make([]disruptor.Consumer, 0, svr.options.NumLoops)
//	for _, loop := range svr.subLoops {
//		ec := &eventConsumer{svr.options.NumLoops, svr.options.NumLoops - 1, loop, connRingBuffer}
//		eventConsumers = append(eventConsumers, ec)
//	}
//
//	// Initialize go-disruptor with ring-buffer for dispatching events to loops.
//	controller := disruptor.Configure(connRingBufferSize).WithConsumerGroup(eventConsumers...).Build()
//
//	controller.Start()
//	defer controller.Stop()
//
//	writer := controller.Writer()
//	sequence := disruptor.InitialSequenceValue
//
//	svr.mainLoop.Run()
//}
