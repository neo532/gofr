package gofunc

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestWithTimeout(t *testing.T) {

	c, closeFn := context.WithCancel(context.Background())

	fn := func(i int) (err error) {
		time.Sleep(time.Second * 1)
		fmt.Println(fmt.Sprintf("%s\t:Biz run,%d", t.Name(), i))
		//err = errors.New("aaaaaaa")
		return
	}

	log := &DefaultLogger{}
	gofn := NewGoFunc(WithLogger(log), WithMaxGoroutine(20))

	l := 3
	fns := make([]func(i int) error, 0, l)
	for i := 0; i < l; i++ {
		fns = append(fns, fn)
	}

	gofn.Go(c, fns...)

	go func() {
		go func() {
			gofn.WithTimeout(
				c,
				time.Second*5,
				fns...,
			)
			err := log.Err()
			fmt.Println(fmt.Sprintf("Print err:<<<%+v>>>", err))
		}()
		select {
		case <-c.Done():
			return
		}
		fmt.Println("Exec Finish")
	}()

	time.Sleep(4 * time.Second)
	fmt.Println("End")
	closeFn()
}

func TestWithTimeoutInN(t *testing.T) {

	fn := func(i int) (err error) {
		time.Sleep(time.Second * 3)
		fmt.Println(fmt.Sprintf("%s\t:Biz run,%d", t.Name(), i))
		//err = errors.New("aaaaaaa")
		return
	}

	log := &DefaultLogger{}
	gofn := NewGoFunc(WithLogger(log), WithMaxGoroutine(20))

	l := 5
	fns := make([]func(i int) error, 0, l)
	for i := 0; i < l; i++ {
		fns = append(fns, fn)
	}

	c, closeFn := context.WithCancel(context.Background())

	gofn.WithTimeout(
		c,
		time.Second*1,
		fns...,
	)
	err := log.Err()
	fmt.Println(fmt.Sprintf("Print err:<<<%+v>>>", err))
	closeFn()
}
