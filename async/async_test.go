package async_test

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/davidbyttow/gosync/async"
	"github.com/stretchr/testify/require"
)

func TestForEach(t *testing.T) {
	t.Parallel()
	t.Run("no op", func(t *testing.T) {
		t.Parallel()
		fn := func() {
			async.ForEach(nil, func(i int, v *int) {
				panic("not handled")
			})
		}
		require.NotPanics(t, fn)
	})
	t.Run("mutates input", func(t *testing.T) {
		t.Parallel()
		input := []int{1, 2, 3, 4, 5}
		async.ForEach(input, func(i int, v *int) {
			*v = *v * i
		})
		require.Equal(t, []int{0, 2, 6, 12, 20}, input)
	})
	t.Run("handles panic", func(t *testing.T) {
		t.Parallel()
		fn := func() {
			async.ForEach([]int{0}, func(i int, v *int) {
				panic("error")
			})
		}
		require.Panics(t, fn)
	})
}

func TestIterator(t *testing.T) {
	t.Parallel()
	t.Run("is concurrent", func(t *testing.T) {
		t.Parallel()
		limit := 50
		iter := async.Iterator[int]{MaxWorkers: limit}
		input := make([]int, limit)
		done := make(chan struct{})
		var count atomic.Int64
		iter.ForEach(input, func(i int, v *int) {
			defer count.Add(-1)
			if int(count.Add(1)) == limit {
				close(done)
				done = nil
			} else {
				<-done
			}
			*v = 1
		})
		require.Equal(t, 1, input[0])
	})
}

func TestPool(t *testing.T) {
	t.Parallel()

	t.Run("panics", func(t *testing.T) {
		t.Parallel()
		pool := async.NewPool()
		pool.Go(func() {
			panic("oops")
		})
		require.Panics(t, pool.Wait)
	})
	t.Run("is concurrent", func(t *testing.T) {
		t.Parallel()
		pool := async.NewPool()

		done := make(chan struct{})
		concurrency := 10
		var count atomic.Int64
		for i := 0; i < concurrency; i++ {
			pool.Go(func() {
				count.Add(1)
				<-done
			})
		}
		for {
			if int(count.Load()) == concurrency {
				close(done)
				break
			}
		}
	})

	t.Run("is limited", func(t *testing.T) {
		t.Parallel()

		max := 5
		pool := async.NewPool().WithMaxWorkers(max)

		concurrency := 10
		var count atomic.Int64
		var over atomic.Int64
		for i := 0; i < concurrency; i++ {
			pool.Go(func() {
				defer count.Add(-1)
				if int(count.Add(1)) > max {
					over.Add(1)
				}
				time.Sleep(time.Millisecond * 5)
			})
		}
		pool.Wait()
		require.Equal(t, 0, int(count.Load()))
		require.Equal(t, 0, int(over.Load()))
	})
}

func TestErrorPool(t *testing.T) {
	t.Parallel()

	err1 := errors.New("one")
	err2 := errors.New("two")

	t.Run("handles one error", func(t *testing.T) {
		t.Parallel()
		pool := async.NewPool().WithErrors()
		pool.Go(func() error {
			return err1
		})
		err := pool.Wait()
		require.Equal(t, "one", err.Error())
	})

	t.Run("handles multiple errors", func(t *testing.T) {
		t.Parallel()
		pool := async.NewPool().WithErrors()
		pool.Go(func() error {
			return err1
		})
		pool.Go(func() error {
			return err2
		})
		err := pool.Wait()
		require.Equal(t, "one\ntwo", err.Error())
	})
}
