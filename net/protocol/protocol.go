package protocol

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
