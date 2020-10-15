# zput_net_golang

like one loop per goroutine thread

- 暂时使用标准的log，
- loop --> loopCtrl --> epoll/kqueue
      ---> event
-       
      
      
      
有的时候确定目标，就行了，不要看到其他更好的实现就开始想实现另外的。   
   
   
  
//TODO超时关闭
func (this *TcpConnect) closeTimeoutConn() func() {
	return func() {
		now := time.Now()
		intervals := now.Sub(time.Unix(this.activeTime.Get(), 0))
		if intervals >= this.idleTime {
			_ = this.Close()
		} else {
			this.timingWheel.AfterFunc(this.idleTime-intervals, this.closeTimeoutConn())
		}
	}
}   
   
   
                drawin kqueue
  ring buffer
  time task
   
  compare standard net/others
  
  unit test
  简化loop
   
  ci/cd
   