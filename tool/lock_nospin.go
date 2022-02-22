package tool

/*
 * @abstract NoSpinLock is a lock without spinning.
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-10-05
 */

import "sync/atomic"

// NoSpinLock is a lock without spinning.
type NoSpinLock struct {
	Done uint32
}

// Lock returns the result of locking.
func (l *NoSpinLock) Lock() bool {
	return atomic.CompareAndSwapUint32(&l.Done, 0, 1)
}

// Unlock returns the result of unlocking.
func (l *NoSpinLock) Unlock() bool {
	return atomic.CompareAndSwapUint32(&l.Done, 1, 0)
}
