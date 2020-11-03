package event_loop

import (
	"github.com/zput/zput_net_golang/net/log"
	"golang.org/x/sys/unix"
	"net"
	"os"
	"testing"
	"time"
)

func TestLoop(t *testing.T){
	log.SetLevel(log.LevelDebug)
	var (
		sign chan bool
		addr string = "127.0.0.1:58866"
		getStringFromLoop string
	)
	const stringToLoop = "hello"
	loop, err := New(1)
	if err != nil {
		t.Fatalf("create fd error; error[%+v]", err)
	}
	var fd = int(newWakeFd(sign, addr, t).Fd())
	var timerEvent = NewEvent(loop, fd)
	timerEvent.SetReadFunc(func(){
		log.Debug("triger read")

		nfd, _, err := unix.Accept(timerEvent.GetFd())
		if err != nil {
			if err != unix.EAGAIN {
				log.Error("accept:", err)
			}
			return
		}

		buf := make([]byte, 5)
		n, err := unix.Read(nfd, buf)
		if err != nil {
			t.Fatalf("fd[%d] read error[%+v]", nfd, err)
		}
		getStringFromLoop = string(buf)
		log.Debugf("triger read[%d] END", n)
	})
	//timerEvent.SetCloseFunc(func(){
	//	log.Debug("triger read")
	//	buf := make([]byte, 10)
	//	n, err := unix.Read(timerEvent.GetFd(), buf)
	//	if n != 0 || err != io.EOF {
	//		t.Fatal()
	//	}
	//})

	go func (){
		select {
		case <-sign:
			return
		default:
			log.Info("enter")
			err := loop.AddEvent(timerEvent)
			if err != nil {
				t.Fatalf("create fd error; error[%+v]", err)
			}
			err = timerEvent.EnableReading(true)
			if err != nil {
				t.Fatalf("create fd error; error[%+v]", err)
			}
			loop.Run()
		}
	}()
	time.Sleep(time.Second)

	conn, err := net.DialTimeout("tcp", addr, time.Second*60)
	if err != nil {
		t.Error(err)
		return
	}

	n, err := conn.Write([]byte(stringToLoop))
	if err != nil {
		t.Error(err)
		return
	}

	log.Infof("--- write over; number[%d]", n)

	//conn.Close()
	//log.Info("--- close over")

	//sign <- true
	time.Sleep(time.Second*3)

	if getStringFromLoop != stringToLoop{
		t.Fatalf("expect %s, get %s", stringToLoop, getStringFromLoop)
	}

	loop.Stop()
}

func newWakeFd(sign <-chan bool, addr string, t *testing.T)(f *os.File){
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("create fd error; error[%+v]", err)
	}

	tcpListener, ok := listener.(*net.TCPListener)
	if !ok {
		t.Fatal("create fd error")
	}
	file, err := tcpListener.File()
	if !ok {
		t.Fatal("create fd error")
	}
	return file
}
