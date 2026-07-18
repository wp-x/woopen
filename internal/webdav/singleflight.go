package webdav

import "sync"

// singleGroup 是最小化的 singleflight：并发相同 key 的调用只执行一次 fn，
// 其余调用等待并复用同一结果。避免 WebDAV 突发请求把联通 API 打爆。
// ponytail: 仅覆盖本包所需，无需引入 golang.org/x/sync。
type singleGroup struct {
	mu sync.Mutex
	m  map[string]*sgCall
}

type sgCall struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// Do 执行 fn，合并同 key 的并发调用。第三个返回值表示结果是否来自共享调用。
func (g *singleGroup) Do(key string, fn func() (interface{}, error)) (interface{}, error, bool) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*sgCall)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err, true
	}
	c := &sgCall{}
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	// 用 defer 保证即便 fn panic，也会释放等待者并清理 key，
	// 否则该 key 会永久留在 map 且 wg 不归零，后续同 key 请求全部死锁。
	func() {
		defer c.wg.Done()
		defer func() {
			g.mu.Lock()
			delete(g.m, key)
			g.mu.Unlock()
		}()
		c.val, c.err = fn()
	}()

	return c.val, c.err, false
}
