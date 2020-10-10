# zput_net_golang

like one loop per goroutine thread

- 暂时使用标准的log，
- loop --> loopCtrl --> epoll/kqueue
      ---> event
-       
      