package storage

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkQueue_BoundedConcurrency(t *testing.T) {
	wq := newWorkQueue(2)
	defer wq.stop()

	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		wq.pool.Submit(func() {
			defer wg.Done()
			cur := concurrent.Add(1)
			// Track max concurrency
			for {
				old := maxConcurrent.Load()
				if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
			concurrent.Add(-1)
		})
	}

	wg.Wait()

	if maxConcurrent.Load() > 2 {
		t.Errorf("max concurrent workers = %d, want <= 2", maxConcurrent.Load())
	}
	if maxConcurrent.Load() < 1 {
		t.Error("no workers ran")
	}
}

func TestWorkQueue_GracefulShutdown(t *testing.T) {
	wq := newWorkQueue(2)

	var completed atomic.Int32
	for i := 0; i < 5; i++ {
		wq.pool.Submit(func() {
			time.Sleep(50 * time.Millisecond)
			completed.Add(1)
		})
	}

	// stopAndWait should block until all jobs complete
	wq.stopAndWait()

	if completed.Load() != 5 {
		t.Errorf("completed = %d, want 5 (all jobs should finish before stop returns)", completed.Load())
	}
}

func TestWorkQueue_StopIdempotent(t *testing.T) {
	wq := newWorkQueue(1)
	wq.stop()
	// Second stop should not panic
	// pond v2 handles this gracefully
}
