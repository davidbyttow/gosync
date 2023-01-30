package sync_test

import (
	"testing"

	"github.com/davidbyttow/gosync/async"
	"github.com/davidbyttow/gosync/sync"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
)

func TestList(t *testing.T) {
	t.Parallel()
	t.Run("appends", func(t *testing.T) {
		t.Parallel()
		var list sync.List[int]
		var expected []int
		count := 100
		var wg async.WaitGroup
		for i := 0; i < count; i++ {
			i := i
			expected = append(expected, i)
			wg.Go(func() {
				list.Append(i)
			})
		}
		wg.Wait()
		actual := list.Slice()
		slices.Sort(actual)
		require.Equal(t, expected, actual)
	})
}

func TestMap(t *testing.T) {
	t.Parallel()
	t.Run("gets", func(t *testing.T) {
		t.Parallel()
		var m sync.Map[string, int]
		v, found := m.Get("key1")
		require.Equal(t, 0, v)
		require.False(t, found)

		v, existed := m.GetOrSet("key1", 42)
		require.Equal(t, 42, v)
		require.False(t, existed)

		v, existed = m.GetOrSet("key1", 50)
		require.Equal(t, 42, v)
		require.True(t, existed)

		v, existed = m.GetOrLoad("key2", func() int { return 50 })
		require.Equal(t, 50, v)
		require.False(t, existed)

		v, existed = m.GetOrLoad("key2", func() int { return 60 })
		require.Equal(t, 50, v)
		require.True(t, existed)
	})
	t.Run("set and ranges", func(t *testing.T) {
		t.Parallel()
		var m sync.Map[int, int]
		for i := 0; i < 10; i++ {
			m.Set(i, i+1)
		}

		var count int
		m.Range(func(k int, v int) bool {
			count++
			require.Equal(t, v, k+1)
			return true
		})
		require.Equal(t, 10, count)

		count = 0
		m.Range(func(k int, v int) bool {
			count++
			require.Equal(t, v, k+1)
			return false
		})
		require.Equal(t, 1, count)
	})
}
