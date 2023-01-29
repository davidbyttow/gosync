package sync

import (
	"sync"
)

type List[T any] struct {
	lock sync.RWMutex
	list []T
}

func (p *List[T]) Append(v T) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.list = append(p.list, v)
}

func (p *List[T]) Slice() []T {
	p.lock.RLock()
	defer p.lock.RUnlock()
	dst := make([]T, len(p.list))
	copy(dst, p.list)
	return dst
}

type Map[Key comparable, T any] struct {
	m    sync.Map
	lock sync.Mutex
}

func (p *Map[Key, T]) Set(k Key, v T) {
	p.m.Store(k, v)
}

func (p *Map[Key, T]) Get(k Key) (T, bool) {
	if v, found := p.m.Load(k); found {
		return v.(T), true
	}
	var zero T
	return zero, false
}

func (p *Map[Key, T]) GetOrSet(k Key, v T) (T, bool) {
	actual, loaded := p.m.LoadOrStore(k, v)
	return actual.(T), loaded
}

func (p *Map[Key, T]) GetOrLoad(k Key, fn func() T) (T, bool) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if v, found := p.m.Load(k); found {
		return v.(T), true
	}
	v := fn()
	p.m.Store(k, v)
	return v, false
}

func (p *Map[Key, T]) Range(fn func(k Key, v T) bool) {
	p.m.Range(func(k any, v any) bool {
		return fn(k.(Key), v.(T))
	})
}
