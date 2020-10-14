package tcpserver

import (
	"github.com/Allenxuxu/gev/log"
	"github.com/Allenxuxu/toolkit/sync"
	"github.com/zput/zput_net_golang/net/event_loop"
	"github.com/zput/zput_net_golang/net/protocol"
	"github.com/zput/zput_net_golang/net/tcpaccept"
	"github.com/zput/zput_net_golang/net/tcpconnect"
	"golang.org/x/sys/unix"
	"runtime"
)

type TcpServer struct{
	options *protocol.Options
	handleEvent IHandleEvent
	loop *event_loop.EventLoop
	threadPoolIsRunLoopFun []*event_loop.EventLoop
	tcpAccept *tcpaccept.TcpAccept
	connectPool map[string]*tcpconnect.TcpConnect
	nextLoopIndex int
}

func New(handleEvent IHandleEvent, loop *event_loop.EventLoop, opts ...protocol.Option)(*TcpServer, error){
	var tcpServer = TcpServer{
		handleEvent:handleEvent,
		loop:loop,
		options:protocol.NewOptions(opts...),
		connectPool:make(map[string]*tcpconnect.TcpConnect),
	}

	var err error
	//创建一个tcp accept
	tcpServer.tcpAccept, err = tcpaccept.New(tcpServer.options.GetNet(), loop)
	if err != nil{
		// TODO log
		return nil, err
	}

	//设置有连接到来后,的回调函数.
	tcpServer.tcpAccept.SetNewConnectCallback(tcpServer.newConnected)

	if tcpServer.options.NumLoops <= 0 {
		tcpServer.options.NumLoops = runtime.NumCPU()
	}

	runloops := make([]*event_loop.EventLoop, tcpServer.options.NumLoops)
	for i := 0; i < tcpServer.options.NumLoops; i++ {
		//这里只是创建结构体，还没有触发开始运行。
		l, err := event_loop.New()
		if err != nil {
			for j := 0; j < i; j++ {
				// How to close?
				_ = runloops[j].Stop()
			}
			return nil, err
		}
		runloops[i] = l
	}
	tcpServer.threadPoolIsRunLoopFun = runloops

	return &tcpServer, nil
}

// Start 启动 Server
func (this *TcpServer) Start() {
	sw := sync.WaitGroupWrapper{}

	length := len(this.threadPoolIsRunLoopFun)
	for i := 0; i < length; i++ {
		sw.AddAndRun(this.threadPoolIsRunLoopFun[i].Run)
	}

	sw.AddAndRun(this.loop.Run)
	sw.Wait()
}

// 停止系统。
func (this *TcpServer) Stop() {
	//先关闭tcpaccept, tcpconnect，然后再关闭loop
	var (
		err error
	)
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
	err = this.loop.Stop()
	if err != nil{
		log.Error(err)
	}
	//关闭connect loop
	for index := range this.threadPoolIsRunLoopFun{
		err = this.threadPoolIsRunLoopFun[index].Stop()
		if err != nil{
			log.Error(err)
		}
	}
}

func (this *TcpServer)newConnected(fd int, sa unix.Sockaddr){
	loopTemp := this.getOneLoopFromPool()

	c, err := tcpconnect.New(loopTemp, fd, sa)
	if err != nil{
		log.Error("failure to create new connection")
		return
	}
	this.addConnect(c.PeerAddr(), c)
	c.SetMessageCallback(this.handleEvent.MessageCallback)
	c.SetConnectCloseCallback(this.connectCloseEvent)
	c.SetWriteCompleteCallback(this.handleEvent.WriteCompletCallback)
	c.ConnectedHandle()

	this.handleEvent.ConnectCallback(c)
}

func (this *TcpServer) getOneLoopFromPool() *event_loop.EventLoop {
	// TODO hash?
	loop := this.threadPoolIsRunLoopFun[this.nextLoopIndex]
	this.nextLoopIndex = (this.nextLoopIndex + 1) % len(this.threadPoolIsRunLoopFun)
	return loop
}


func (this *TcpServer) connectCloseEvent(connect *tcpconnect.TcpConnect){
	this.handleEvent.ConnectCloseCallback(connect)
	this.removeConnect(connect.PeerAddr())
}

func (this *TcpServer) addConnect(name string, connect *tcpconnect.TcpConnect) {
	this.connectPool[name] = connect
}

func (this *TcpServer) removeConnect(name string){
	_, ok := this.connectPool[name]
	if ok {
		delete(this.connectPool, name)
	}
}
