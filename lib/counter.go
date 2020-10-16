/*
 * @abstract ths auto incrementer
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-09-26
 */
package lib

import "runtime"

// NewAutoInc is a instance for Counter.
type AutoInc struct {
	start uint64
	step  uint64
	bRun  bool
	queue chan uint64
}

// NewAutoInc return a instance of AutoInc.
func NewAutoInc(iStart, iStep uint64) *AutoInc {
	ai := &AutoInc{
		start: iStart,
		step:  iStep,
		bRun:  true,
		queue: make(chan uint64, runtime.NumCPU()),
	}
	go ai.set()
	return ai
}

func (c *AutoInc) set() {
	defer func() { recover() }()
	for i := c.start; c.bRun; i = i + c.step {
		c.queue <- i
	}
}

// Get return a uint64 of counter.
func (c *AutoInc) Get() uint64 {
	return <-c.queue
}

// Close close the counter.
func (c *AutoInc) Close() {
	c.bRun = false
	close(c.queue)
}
