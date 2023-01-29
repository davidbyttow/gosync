package safe_test

import (
	"testing"

	"github.com/davidbyttow/gosync/safe"
	"github.com/stretchr/testify/require"
)

func TestTry(t *testing.T) {
	t.Run("panics", func(t *testing.T) {
		res := safe.Try(func() {
			panic("fail")
		})
		require.NotNil(t, res)
	})

	t.Run("no panic", func(t *testing.T) {
		res := safe.Try(func() {})
		require.Nil(t, res)
	})
}
