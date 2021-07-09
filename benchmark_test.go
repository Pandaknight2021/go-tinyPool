package tinyPool

import (
	"sync"
	"testing"
	"time"
)

const (
	RunTimes           = 1000000
	BenchParam         = 10
	PoolSize           = 5000
	DefaultExpiredTime = 10 * time.Second
	RunTimesL          = 100000
)

func Fib(n int) int {
	if n < 3 {
		return 1
	}
	d := [3]int{1, 1, 0}
	for i := 2; i < n; i++ {
		d[2] = d[1] + d[0]
		d[0], d[1] = d[1], d[2]
	}
	return d[2]
}

func slow() {
	for t0 := time.Now(); time.Since(t0) < 1e7; {
		Fib(1000)
		time.Sleep(time.Microsecond)
	}
}

func fast() {
	Fib(1000)
	time.Sleep(10 * time.Millisecond)
}

func BenchmarkGoroutines_fast(b *testing.B) {
	var wg sync.WaitGroup
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(RunTimes)
		for j := 0; j < RunTimes; j++ {
			go func() {
				fast()
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

func BenchmarkTinyPool_fast(b *testing.B) {
	var wg sync.WaitGroup
	p, _ := NewPool(PoolSize)
	defer p.Close()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(RunTimes)
		for j := 0; j < RunTimes; j++ {
			_ = p.Submit(func() {
				fast()
				wg.Done()
			})
		}
		wg.Wait()
	}
	b.StopTimer()
}

func BenchmarkGoroutines_slow(b *testing.B) {
	var wg sync.WaitGroup

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(RunTimesL)
		for j := 0; j < RunTimesL; j++ {
			go func() {
				slow()
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

func BenchmarkTinyPool_slow(b *testing.B) {
	var wg sync.WaitGroup
	p, _ := NewPool(PoolSize)
	defer p.Close()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(RunTimesL)
		for j := 0; j < RunTimesL; j++ {
			_ = p.Submit(func() {
				slow()
				wg.Done()
			})
		}
		wg.Wait()
	}
	b.StopTimer()
}
