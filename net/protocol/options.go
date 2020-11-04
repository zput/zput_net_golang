package protocol

import (
	"time"
)

// Options 服务配置
type Options struct {
	net      NetWorkAndAddressAndOption
	NumLoops int

	tick      time.Duration
	wheelSize int64
	IdleTime  time.Duration

	codeImp ICode
}

// Option ...
type Option func(*Options)

func(this *Options)GetNet() NetWorkAndAddressAndOption {
	return this.net
}

func(this *Options)GetTick() time.Duration {
	return this.tick
}

func(this *Options)GetWheelSize() int64 {
	return this.wheelSize
}

func(this *Options)GetCode() ICode {
	return this.codeImp
}

func NewOptions(opt ...Option) *Options {
	opts := Options{}

	for _, o := range opt {
		o(&opts)
	}

	if len(opts.net.Network) == 0 {
		opts.net.Network = "tcp"
	}
	if len(opts.net.Address) == 0 {
		opts.net.Address = ":58800"
	}
	if opts.tick == 0 {
		opts.tick = 1 * time.Millisecond
	}
	if opts.wheelSize == 0 {
		opts.wheelSize = 1000
	}
	if opts.codeImp == nil{
		// TODO
		opts.codeImp = new(DefaultCode)
	}

	return &opts
}

// ReusePort 设置 SO_REUSEPORT
func ReusePort(reusePort bool) Option {
	return func(o *Options) {
		o.net.ReusePort = reusePort
	}
}

func Network(n string) Option {
	return func(o *Options) {
		o.net.Network = n
	}
}

// Address server 监听地址
func Address(a string) Option {
	return func(o *Options) {
		o.net.Address = a
	}
}

// NumLoops work eventloop 的数量
func NumLoops(n int) Option {
	return func(o *Options) {
		o.NumLoops = n
	}
}

// IdleTime 最大空闲时间（秒）
func IdleTime(t time.Duration) Option {
	return func(o *Options) {
		o.IdleTime = t
	}
}

func CodeImp(codeImp ICode) Option {
	return func(o *Options) {
		o.codeImp = codeImp
	}
}
