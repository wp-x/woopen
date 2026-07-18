package webdav

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestSingleGroupDedup 验证并发相同 key 只执行一次 fn，其余复用结果。
func TestSingleGroupDedup(t *testing.T) {
	var g singleGroup
	var calls int32
	start := make(chan struct{})

	const n = 20
	var wg sync.WaitGroup
	results := make([]interface{}, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start // 所有 goroutine 同时冲入 Do，保证重叠
			v, _, _ := g.Do("k", func() (interface{}, error) {
				atomic.AddInt32(&calls, 1)
				time.Sleep(80 * time.Millisecond) // 停留在飞行中，让其余调用挂起等待
				return 42, nil
			})
			results[idx] = v
		}(i)
	}

	close(start)
	wg.Wait()

	if calls != 1 {
		t.Fatalf("fn 应只执行一次，实际 %d 次", calls)
	}
	for i := 0; i < n; i++ {
		if results[i] != 42 {
			t.Fatalf("结果不一致: %v", results[i])
		}
	}

	// 上一批完成后，新调用应重新执行 fn（key 已释放）。
	v, _, shared := g.Do("k", func() (interface{}, error) { return 7, nil })
	if v != 7 || shared {
		t.Fatalf("批次结束后应重新执行, v=%v shared=%v", v, shared)
	}
}

// TestSingleGroupPanicCleansUp 确保 fn panic 后 key 被清理，不会永久死锁。
func TestSingleGroupPanicCleansUp(t *testing.T) {
	var g singleGroup

	func() {
		defer func() { _ = recover() }()
		g.Do("p", func() (interface{}, error) { panic("boom") })
	}()

	// panic 后同 key 再次调用必须能正常执行（若 key 未清理，这里会永久阻塞）。
	done := make(chan struct{})
	go func() {
		v, _, _ := g.Do("p", func() (interface{}, error) { return 1, nil })
		if v != 1 {
			t.Errorf("panic 后重试结果错误: %v", v)
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("panic 后同 key 调用死锁，未清理 key")
	}
}
