package async_test

import (
	"errors"
	"strings"
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
		var total atomic.Int64
		async.ForEach(input, func(i int, v int) {
			total.Add(int64(v))
		})
		require.Equal(t, 15, int(total.Load()))
	})
	t.Run("handles panic", func(t *testing.T) {
		t.Parallel()
		fn := func() {
			async.ForEach([]int{0}, func(i int, v int) {
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
		iter.ForEach(input, func(i int, v int) {
			defer count.Add(-1)
			if int(count.Add(1)) == limit {
				close(done)
				done = nil
			} else {
				<-done
			}
		})
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

func TestResultPool(t *testing.T) {
	t.Parallel()

	err1 := errors.New("one")
	err2 := errors.New("two")

	t.Run("returns values", func(t *testing.T) {
		t.Parallel()
		pool := async.NewResultPool[int]()
		for i := 0; i < 3; i++ {
			i := i
			pool.Go(func() (int, error) {
				return i, nil
			})
		}
		res, err := pool.Wait()
		require.ElementsMatch(t, res, []int{0, 1, 2})
		require.NoError(t, err)
	})
	t.Run("handles errors", func(t *testing.T) {
		t.Parallel()
		pool := async.NewResultPool[int]()
		for i := 0; i < 4; i++ {
			i := i
			pool.Go(func() (int, error) {
				if i == 1 {
					return 0, err1
				} else if i == 3 {
					return 0, err2
				}
				return i, nil
			})
		}
		res, err := pool.Wait()
		require.ElementsMatch(t, res, []int{0, 2})
		require.ElementsMatch(t, strings.Split(err.Error(), "\n"), []string{"one", "two"})
	})
}
