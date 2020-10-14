package protocol

import (
	"errors"
	"golang.org/x/sys/unix"
)

const PollTimeMs = 3000

//using DefaultFunction = std::function<void()>
type DefaultFunction func()

// Event poller 返回事件
type EventType uint32

// Event poller 返回事件值
const (
	EventRead  EventType = 0x1
	EventWrite EventType = 0x2
	EventErr   EventType = 0x80
	EventClose   EventType = 0x100
	EventNone  EventType = 0
)

//handler func(fd int, event Event)
type EmbedHandler2Multiplex func(fd int, eventType EventType)

// The network must be "tcp", "tcp4", "tcp6", "unix" or "unixpacket".
type NetWorkAndAddressAndOption struct {
	Network, Address string
	ReusePort bool
}

// TCP accept 处理新连接
type OnNewConnectCallback func(fd int, sa unix.Sockaddr)

type AddFunToLoopWaitingRun func()

// ErrClosed 重复 close poller 错误
var ErrClosed = errors.New("poller instance is not running")
