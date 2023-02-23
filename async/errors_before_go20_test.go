//go:build !go1.20

package async

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type multiError interface {
	Unwrap() []error
}

func TestJoinErrors(t *testing.T) {
	t.Parallel()

	err1 := errors.New("one")
	err2 := errors.New("two")
	err3 := errors.New("three")

	t.Run("handles nil", func(t *testing.T) {
		require.Equal(t, &joinError{errs: []error{err1, err2}}, joinErrors(err1, nil, err2))
	})
	t.Run("satisfies multi-error", func(t *testing.T) {
		err := joinErrors(err1, err2)
		me, ok := err.(multiError)
		require.True(t, ok)
		require.Equal(t, []error{err1, err2}, me.Unwrap())
	})
	t.Run("concatenates strings", func(t *testing.T) {
		err := joinErrors(err1, err2, err3)
		require.Equal(t, "one\ntwo\nthree", err.Error())
	})
}
