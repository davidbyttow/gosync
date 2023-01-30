package async

import (
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/davidbyttow/gosync/safe"
	gosync "github.com/davidbyttow/gosync/sync"
)

func ForEach[T any](items []T, fn func(int, T)) {
	forEach(items, fn, 0)
}

type Iterator[T any] struct {
	MaxWorkers int
}

func (iter Iterator[T]) ForEach(items []T, fn func(int, T)) {
	forEach(items, fn, iter.MaxWorkers)
}

func forEach[T any](items []T, fn func(int, T), numWorkers int) {
	if numWorkers == 0 {
		numWorkers = runtime.GOMAXPROCS(0)
	}
	numItems := len(items)
	if numWorkers > numItems {
		numWorkers = numItems
	}

	var cur atomic.Int64
	proc := func() {
		next := cur.Add(1) - 1
		for next < int64(numItems) {
			fn(int(next), items[next])
			next = cur.Add(1) - 1
		}
	}

	var wg WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Go(proc)
	}
	wg.Wait()
}

type Pool struct {
	once   sync.Once
	wg     WaitGroup
	tasks  chan func()
	waiter chan struct{}
}

func NewPool() *Pool {
	return &Pool{}
}

func (p *Pool) WithMaxWorkers(limit int) *Pool {
	if limit < 1 {
		panic("limit must be greater than 0")
	}
	p.waiter = make(chan struct{}, limit)
	return p
}

func (p *Pool) WithErrors() *ErrorPool {
	return &ErrorPool{pool: p}
}

func (p *Pool) Go(fn func()) {
	p.once.Do(func() {
		p.tasks = make(chan func())
	})

	if p.waiter == nil {
		select {
		case p.tasks <- fn:
		default:
			p.wg.Go(p.process)
			p.tasks <- fn
		}
		return
	}

	select {
	case p.waiter <- struct{}{}:
		p.wg.Go(p.process)
		p.tasks <- fn
	case p.tasks <- fn:
	}
}

func (p *Pool) Wait() {
	if p.tasks != nil {
		close(p.tasks)
		p.tasks = nil
	}
	p.wg.Wait()
}

func (p *Pool) process() {
	if p.waiter != nil {
		defer func() {
			<-p.waiter
		}()
	}
	for fn := range p.tasks {
		fn()
	}
}

type ErrorPool struct {
	pool *Pool
	err  error
	lock sync.Mutex
}

func (p *ErrorPool) Go(fn func() error) {
	p.pool.Go(func() {
		err := fn()
		p.maybeAddError(err)
	})
}

func (p *ErrorPool) Wait() error {
	p.pool.Wait()
	return p.err
}

func (p *ErrorPool) maybeAddError(err error) {
	if err == nil {
		return
	}
	p.lock.Lock()
	defer p.lock.Unlock()
	p.err = joinErrors(p.err, err)
}

type ResultPool[T any] struct {
	pool    *ErrorPool
	results gosync.List[T]
}

func NewResultPool[T any]() *ResultPool[T] {
	return &ResultPool[T]{pool: NewPool().WithErrors()}
}

func (p *ResultPool[T]) Go(fn func() (T, error)) {
	p.pool.Go(func() error {
		res, err := fn()
		if err == nil {
			p.results.Append(res)
		}
		return err
	})
}

func (p *ResultPool[T]) Wait() ([]T, error) {
	err := p.pool.Wait()
	return p.results.Slice(), err
}

type WaitGroup struct {
	wg  sync.WaitGroup
	try safe.Catcher
}

func (p *WaitGroup) Go(fn func()) {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.try.Try(fn)
	}()
}

func (p *WaitGroup) Wait() {
	p.wg.Wait()
	p.try.Rethrow()
}

func (p *WaitGroup) WaitAndCatch() *safe.RecoveredPanic {
	p.wg.Wait()
	return p.try.Caught()
}

func (p *WaitGroup) WaitErr() error {
	return p.WaitAndCatch().Error()
}
