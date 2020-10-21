package protocol

import "sync/atomic"

type Bool struct {
	b int32
}

func (a *Bool) Set(b bool) bool {
	var newV int32
	if b {
		newV = 1
	}
	return atomic.SwapInt32(&a.b, newV) == 1
}

func (a *Bool) Get() bool {
	return atomic.LoadInt32(&a.b) == 1
}

// Int64 提供原子操作
type Int64 struct {
	v int64
}

// Add 计数增加 i ，减操作：Add(-1)
func (a *Int64) Add(i int64) int64 {
	return atomic.AddInt64(&a.v, i)
}

// Swap 交换值，并返回原来的值
func (a *Int64) Swap(i int64) int64 {
	return atomic.SwapInt64(&a.v, i)
}

// Get 获取值
func (a *Int64) Get() int64 {
	return atomic.LoadInt64(&a.v)
}
