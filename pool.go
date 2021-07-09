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

	"github.com/pandaknight2021/queue"
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

	idle int32

	q *queue.MpscQueue

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
		idle:     0,
		q:        queue.NewMpscQueue(),
	}

	go p.dispatch()

	return p, nil
}

func (p *Pool) Submit(task func()) error {
	if p.isClosed {
		return errors.New("pool closed")
	}

	if task != nil {
		running := p.Running()
		if running < p.capacity {
			if atomic.CompareAndSwapInt32(&p.running, running, running+1) {
				p.startOneWorker()
			}
		}

		if idle := atomic.LoadInt32(&p.idle); idle > 0 {
			p.task <- task
		} else {
			p.q.Push(task)
		}

		atomic.AddInt32(&p.jobNum, 1)
	}
	return nil
}

func (p *Pool) dispatch() {
	ticker := time.NewTicker(time.Duration(p.expiry))
	defer ticker.Stop()

	go func() {
		for !p.isClosed {
			if p.q.Size() > 0 {
				task := p.q.Pop()
				p.task <- task.(func())
			} else {
				time.Sleep(10 * time.Microsecond)
			}
		}
	}()

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
	defer p.wg.Done()

	atomic.AddInt32(&p.idle, 1)
	defer atomic.AddInt32(&p.idle, -1)

	defer atomic.AddInt32(&p.running, -1)

	for fn := range p.task {
		if fn != nil {
			atomic.AddInt32(&p.idle, -1)
			fn()
			atomic.AddInt32(&p.idle, 1)
		} else {
			break
		}
	}
}
