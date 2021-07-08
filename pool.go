// MIT License

// Copyright (c) 2021 pandaKnight

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

//Package tinyPool queues work to a limited number of goroutines.
package tinyPool

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// If workes idle for at least this period of time, then stop a worker.
	expireTimeout = 2 * int(time.Second)
)

type Pool struct {
	// capacity of the pool
	capacity int32

	//currently running goroutines
	running int32

	//task queue -> task
	task chan func()

	jobNum int32

	wg sync.WaitGroup

	quitSig chan struct{}

	// expire time for recycle goroutine
	expiry int

	isClosed bool
}

// NewPool generates an instance of pool.
func NewPool(size int) (*Pool, error) {
	cap := runtime.NumCPU()
	if cap < size {
		cap = size
	}

	p := &Pool{
		capacity: int32(cap),
		running:  int32(0),
		task:     make(chan func()),
		quitSig:  make(chan struct{}),
		expiry:   expireTimeout,
		isClosed: false,
		jobNum:   0,
	}

	go p.monitor()

	return p, nil
}

func (p *Pool) Submit(task func(), syncMode bool) error {
	if p.isClosed {
		return errors.New("pool closed")
	}

	if task != nil {
		if syncMode {
			doneChan := make(chan struct{})
			v := func() {
				task()
				close(doneChan)
			}

			running := p.Running()
			if running < p.capacity {
				if atomic.CompareAndSwapInt32(&p.running, running, 1) {
					p.startOneWorker()
				}
			}
			p.task <- v
			atomic.AddInt32(&p.jobNum, 1)
			<-doneChan
		} else {
			running := p.Running()
			if running < p.capacity {
				if atomic.CompareAndSwapInt32(&p.running, running, running+1) {
					p.startOneWorker()
				}
			}
			p.task <- task
			atomic.AddInt32(&p.jobNum, 1)
		}
	}
	return nil
}

func (p *Pool) monitor() {
	ticker := time.NewTicker(time.Duration(p.expiry))
	defer ticker.Stop()

outer:
	for {
		n := p.jobNum
		select {
		case <-p.quitSig:
			break outer

		case <-ticker.C:
			if n == p.jobNum {
				if p.Running() > 0 {
					p.stopOneWorker()
				}
			}
		}
	}
}

func (p *Pool) Close() {
	p.isClosed = true
	close(p.quitSig)
	close(p.task)
	p.wg.Wait()
}

func (p *Pool) Running() int32 {
	return int32(atomic.LoadInt32(&p.running))
}

func (p *Pool) startOneWorker() {
	go p.worker()
}

func (p *Pool) stopOneWorker() {
	p.task <- nil
}

func (p *Pool) worker() {
	p.wg.Add(1)
	defer atomic.AddInt32(&p.running, -1)
	defer p.wg.Done()

	for fn := range p.task {
		if fn != nil {
			fn()
		} else {
			break
		}
	}
}
