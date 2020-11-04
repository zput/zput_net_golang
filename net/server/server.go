package tcpserver

import (
	"github.com/RussellLuo/timingwheel"
	"github.com/zput/ringbuffer"
	"github.com/zput/zput_net_golang/net/event_loop"
	"github.com/zput/zput_net_golang/net/log"
	"github.com/zput/zput_net_golang/net/protocol"
	"github.com/zput/zput_net_golang/net/accept"
	"github.com/zput/zput_net_golang/net/connect"
	"golang.org/x/sys/unix"
	"runtime"
	"time"
)

type Server struct{
	options *protocol.Options
	handleEvent IHandleEvent
	mainLoop *event_loop.EventLoop
	subLoops []*event_loop.EventLoop
	tcpAccept *accept.Accept
	connectPool map[string]*connect.Connect
	nextLoopIndex int

	timingWheel *timingwheel.TimingWheel
}

func New(handleEvent IHandleEvent, opts ...protocol.Option)(*Server, error){
	var err error

	mainLoop, err := event_loop.New(-1)
	if err != nil{
		log.Errorf("new mainLoop error[%v]", err)
		return nil, err
	}

	var tcpServer = Server{
		handleEvent:handleEvent,
		mainLoop:mainLoop,
		options:protocol.NewOptions(opts...),
		connectPool:make(map[string]*connect.Connect),
	}

	tcpServer.timingWheel = timingwheel.NewTimingWheel(tcpServer.options.GetTick(), tcpServer.options.GetWheelSize())

	//创建一个tcp accept
	tcpServer.tcpAccept, err = accept.New(tcpServer.options.GetNet(), tcpServer.mainLoop)
	if err != nil{
		log.Errorf("new accept error[%v]", err)
		return nil, err
	}

	//设置有连接到来后,的回调函数.
	tcpServer.tcpAccept.SetNewConnectCallback(tcpServer.newConnected)

	if tcpServer.options.NumLoops <= 0 {
		if tcpServer.options.NumLoops == 0 {
			tcpServer.options.NumLoops = 1
		} else {
			tcpServer.options.NumLoops = runtime.NumCPU()
		}
	}
	tcpServer.options.NumLoops = ringbuffer.NotMoreThan(tcpServer.options.NumLoops)

	runloops := make([]*event_loop.EventLoop, tcpServer.options.NumLoops)
	for i := 0; i < tcpServer.options.NumLoops; i++ {
		//这里只是创建结构体，还没有触发开始运行。
		l, err := event_loop.New(i)
		if err != nil {
			for j := 0; j < i; j++ {
				// How to close?
				_ = runloops[j].Stop()
			}
			return nil, err
		}
		runloops[i] = l
	}
	tcpServer.subLoops = runloops

	return &tcpServer, nil
}

// Start 启动 Server
func (this *Server) Start() {
	this.timingWheel.Start()

	err := this.tcpAccept.Listen()
	if err != nil{
		panic(err)
	}

	sw := protocol.WaitGroupWrapper{}
	length := len(this.subLoops)
	for i := 0; i < length; i++ {
		sw.AddAndRun(this.subLoops[i].Run)
	}
	sw.AddAndRun(this.mainLoop.Run)



	sw.Wait()
}

// 停止系统。
func (this *Server) Stop() {
	//先关闭tcpaccept, tcpconnect，然后再关闭loop
	var (
		err error
	)

	this.timingWheel.Stop()

	err = this.tcpAccept.Close()
	if err != nil{
		log.Error(err)
	}
	for  k, v := range this.connectPool{
	    err = v.Close()
		if err != nil{
			log.Errorf("closed [%s] failure, error[%v]", k, err)
		}
	}
	//关闭accept AND main loop
	err = this.mainLoop.Stop()
	if err != nil{
		log.Error(err)
	}
	//关闭connect loop
	for index := range this.subLoops{
		err = this.subLoops[index].Stop()
		if err != nil{
			log.Error(err)
		}
	}
}

// RunAfter 延时任务
func (this *Server) RunAfter(d time.Duration, f func()) *timingwheel.Timer {
	return this.timingWheel.AfterFunc(d, f)
}

// RunEvery 定时任务
func (this *Server) RunEvery(d time.Duration, f func()) *timingwheel.Timer {
	return this.timingWheel.ScheduleFunc(&protocol.EveryScheduler{Interval: d}, f)
}

func (this *Server) newConnected(fd int, sa unix.Sockaddr){
	loopTemp := this.getOneLoopFromPool()

	c, err := connect.New(loopTemp, fd, sa, this.timingWheel, this.options.IdleTime, this.options.GetCode())
	if err != nil{
		log.Errorf("failure to create new connection; error[%v]", err)
		return
	}

	log.Debugf("a connection[%s] is enter", c.PeerAddr())

	this.addConnect(c.PeerAddr(), c)
	c.SetMessageCallback(this.handleEvent.MessageCallback)
	c.SetConnectCloseCallback(this.connectCloseEvent)
	c.SetWriteCompleteCallback(this.handleEvent.WriteCompletCallback)
	loopTemp.RunInLoop(func(){
		if err := c.ConnectedHandle(); err != nil{
			c.Close()
			this.removeConnect(c.PeerAddr())
		}
		this.handleEvent.ConnectCallback(c)
	})
}

func (this *Server) getOneLoopFromPool() *event_loop.EventLoop {
	// TODO hash?
	loop := this.subLoops[this.nextLoopIndex]
	this.nextLoopIndex = (this.nextLoopIndex + 1) % len(this.subLoops)
	return loop
}

func (this *Server) connectCloseEvent(connect *connect.Connect){
	this.handleEvent.ConnectCloseCallback(connect)
	this.removeConnect(connect.PeerAddr())
	log.Debug("in server; delete connect pool")
}

func (this *Server) addConnect(name string, connect *connect.Connect) {
	this.connectPool[name] = connect
}

func (this *Server) removeConnect(name string){
	_, ok := this.connectPool[name]
	if ok {
		delete(this.connectPool, name)
	}
}
