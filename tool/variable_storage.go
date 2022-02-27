package tool

import (
	"context"
	"sync/atomic"
	"time"
)

/*
 * @abstract VarStorage is a plan to storage the data in a variable.
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-10-05
 * @example: github.com/neo532/gofr/blob/main/example/tool/variable_test.go
 */

// ========== IVarStorageByLock ==========

// IVarStorageByLock is the interface for VarStorageByLock's functions.
type IVarStorageByLock interface {
	Update(c context.Context) interface{}
	IsVaild(c context.Context) bool
}

// VarStorageByLock is the struct for NewVarStorageByLock.
type VarStorageByLock struct {
	data    atomic.Value
	opFn    IVarStorageByLock
	lock    *NoSpinLock
	timeout time.Duration

	duration time.Duration
	errFn    func(c context.Context, err error)
	ctx      context.Context
}

// ---------- opts ----------

// VSLopt offers some options for VarStorageByLock.
type VSLopt func(*VarStorageByLock)

// OpFun sets the handle of operation for VarStorageByLock.
func (l VSLopt) OpFun(fn IVarStorageByLock) VSLopt {
	return func(v *VarStorageByLock) {
		v.opFn = fn
	}
}

// Duration sets duration for VarStorageByLock.
func (l VSLopt) Duration(t time.Duration) VSLopt {
	return func(v *VarStorageByLock) {
		v.duration = t
	}
}

// Timeout sets timeout for VarStorageByLock.
func (l VSLopt) Timeout(t time.Duration) VSLopt {
	return func(v *VarStorageByLock) {
		v.timeout = t
	}
}

// ErrFun sets the handle of error for VarStorageByLock.
func (l VSLopt) ErrFun(fn func(c context.Context, err error)) VSLopt {
	return func(v *VarStorageByLock) {
		v.errFn = fn
	}
}

// Context sets the context for VarStorageByLock.
func (l VSLopt) Context(c context.Context) VSLopt {
	return func(v *VarStorageByLock) {
		v.ctx = c
	}
}

// NewVarStorageByLock returns a instance of updating data by locking.
func NewVarStorageByLock(opts ...VSLopt) (l *VarStorageByLock) {
	l = &VarStorageByLock{
		timeout:  0,
		duration: defDuration,
		errFn:    defErrFn,
		ctx:      context.Background(),
	}

	for _, o := range opts {
		o(l)
	}

	l.lock = &NoSpinLock{}
	l.set(l.ctx)

	fn := func(i int) {
		tick := time.Tick(l.duration)
		for {
			select {
			case <-tick:
				l.lock.Unlock()
			}
		}
	}

	var opt GFopt
	NewGoFunc(opt.ErrFunc(l.errFn)).AsyncGo(l.ctx, fn)

	return l
}

func (l *VarStorageByLock) set(c context.Context) {
	l.data.Store(l.opFn.Update(c))
}

// Get returns the data and trigger the thing that updating the data.
func (l *VarStorageByLock) Get(c context.Context) interface{} {
	if l.lock.Lock() {
		if l.opFn.IsVaild(c) == false {
			go func() {
				l.set(c)
			}()
		}
	}

	return l.data.Load()
}

// ========== IVarStorageByTick ==========

// IVarStorageByTick is the interface for VarStorageByTick's function.
type IVarStorageByTick interface {
	Update(c context.Context) interface{}
}

// VarStorageByTick is the struct for NewVarStorageByTick.
type VarStorageByTick struct {
	data     atomic.Value
	opFn     IVarStorageByTick
	duration time.Duration
	timeout  time.Duration

	errFn func(c context.Context, err error)
	ctx   context.Context
}

// ---------- opts ----------

// VSTopt offers some options for VarStorageByTick.
type VSTopt func(*VarStorageByTick)

// OpFun sets the handle of operation for VarStorageByTick.
func (t VSTopt) OpFun(fn IVarStorageByTick) VSTopt {
	return func(v *VarStorageByTick) {
		v.opFn = fn
	}
}

// Duration sets duration for VarStorageByTick.
func (t VSTopt) Duration(s time.Duration) VSTopt {
	return func(v *VarStorageByTick) {
		v.duration = s
	}
}

// Timeout sets timeout for VarStorageByTick.
func (t VSTopt) Timeout(ts time.Duration) VSTopt {
	return func(v *VarStorageByTick) {
		v.timeout = ts
	}
}

// ErrFun sets the handle of error for VarStorageByTick.
func (t VSTopt) ErrFun(fn func(c context.Context, err error)) VSTopt {
	return func(v *VarStorageByTick) {
		v.errFn = fn
	}
}

// Context sets the context for VarStorageByTick.
func (t VSTopt) Context(c context.Context) VSTopt {
	return func(v *VarStorageByTick) {
		v.ctx = c
	}
}

// NewVarStorageByTick returns a instance of updating data by ticking.
func NewVarStorageByTick(opts ...VSTopt) (l *VarStorageByTick) {
	l = &VarStorageByTick{
		duration: defDuration,
		timeout:  defTimeout,
		errFn:    defErrFn,
		ctx:      context.Background(),
	}
	for _, o := range opts {
		o(l)
	}

	l.set()

	fn := func(i int) {
		tick := time.Tick(l.duration)
		for {
			select {
			case <-tick:
				l.set()
			}
		}
	}

	var opt GFopt
	NewGoFunc(opt.ErrFunc(l.errFn)).AsyncGo(l.ctx, fn)

	return l
}

func (l *VarStorageByTick) set() {
	l.data.Store(l.opFn.Update(context.Background()))
}

// Get returns the data.
func (l *VarStorageByTick) Get() interface{} {
	return l.data.Load()
}
