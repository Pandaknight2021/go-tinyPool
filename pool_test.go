package tinyPool

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

const (
	_  = 1 << (10 * iota)
	KB // 1024
	MB // 1048576
)

func TestFib(t *testing.T) {
	t0 := time.Now()
	for j := 0; j < RunTimes; j++ {
		Fib(100)
	}

	t.Logf("fib(100): = %v ", time.Since(t0)/RunTimes)

	t0 = time.Now()
	for j := 0; j < RunTimes; j++ {
		Fib(1000)
	}

	t.Logf("fib(1000): = %v ", time.Since(t0)/RunTimes)
}

func TestTinyPool(t *testing.T) {
	var wg sync.WaitGroup
	p, _ := NewPool(PoolSize)
	defer p.Close()

	t0 := time.Now()

	wg.Add(RunTimes)
	for j := 0; j < RunTimes; j++ {
		_ = p.Submit(func() {
			Fib(100)
			wg.Done()
		})
	}
	t.Logf("push elapsed = %v ", time.Since(t0))
	wg.Wait()

	//time.Sleep(60 * time.Second)

	t.Logf("elapsed = %v ", time.Since(t0))

	m := runtime.MemStats{}
	runtime.ReadMemStats(&m)

	t.Logf("Alloc = %v MiB", (m.Alloc)/MB)
	t.Logf("\tTotalAlloc = %v MiB", (m.TotalAlloc)/MB)
	t.Logf("\tSys = %v MiB", m.Sys/MB)
	t.Logf("\tNumGC = %v", m.NumGC)
	t.Logf("\tAllocObjCnt = %v", m.Mallocs)
	t.Logf("\tSTW = %vms\n", m.PauseTotalNs/1e6)
	t.Logf("\tGCCPUFraction = %v\n", m.GCCPUFraction)
}
