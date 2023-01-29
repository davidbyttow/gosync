package safe

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"sync/atomic"

	pkgErr "github.com/pkg/errors"
)

// Try calls the given function and recovers from any panic.
// If a panic is not raised, the returned value will be nil.
func Try(fn func()) *RecoveredPanic {
	c := Catcher{}
	c.Try(fn)
	return c.Caught()
}

// Catcher handles safely catching the panic. Use `Try` instead of this
// class direction.
type Catcher struct {
	caught atomic.Pointer[RecoveredPanic]
}

// Try calls the given function and recovers from any panic.
// If a panic is not raised, the returned value will be nil.
func (p *Catcher) Try(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			c := catch(r, 1)
			p.caught.CompareAndSwap(nil, &c)
		}
	}()
	fn()
}

func (p *Catcher) Caught() *RecoveredPanic {
	return p.caught.Load()
}

// Rethrow will panic with the caught RecoveredPanic.
// If no panic was recovered then this is a no-op.
func (p *Catcher) Rethrow() {
	if r := p.Caught(); r != nil {
		panic(r)
	}
}

type RecoveredPanic struct {
	Value      any
	DebugTrace []byte
	st         pkgErr.StackTrace
}

func catch(value any, skipFrames int) RecoveredPanic {
	stack := callers(skipFrames)
	return RecoveredPanic{
		Value:      value,
		DebugTrace: debug.Stack(),
		st:         stack.StackTrace(),
	}
}

func (p *RecoveredPanic) String() string {
	return fmt.Sprintf("panic: %v\nstacktrace:\n%s\n", p.Value, p.DebugTrace)
}

// Error returns a RecoveredPanicError
// It is safe to call this on a nil value and the result will be nil.
func (p *RecoveredPanic) Error() error {
	if p == nil {
		return nil
	}
	return &RecoveredPanicError{*p}
}

type RecoveredPanicError struct {
	RecoveredPanic
}

var _ error = (*RecoveredPanicError)(nil)

func (p *RecoveredPanicError) Error() string { return p.String() }
func (p *RecoveredPanicError) Unwrap() error {
	if err, ok := p.Value.(error); ok {
		return err
	}
	return nil
}
func (p *RecoveredPanicError) StackTrace() pkgErr.StackTrace {
	return p.st
}

type StackTraceProvider interface {
	StackTrace() pkgErr.StackTrace
}

type stack []uintptr

func (s *stack) Format(st fmt.State, verb rune) {
	switch verb {
	case 'v':
		switch {
		case st.Flag('+'):
			for _, pc := range *s {
				f := pkgErr.Frame(pc)
				fmt.Fprintf(st, "\n%+v", f)
			}
		}
	}
}

func (s *stack) StackTrace() pkgErr.StackTrace {
	f := make([]pkgErr.Frame, len(*s))
	for i := 0; i < len(f); i++ {
		f[i] = pkgErr.Frame((*s)[i])
	}
	return f
}

func callers(skip int) stack {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(skip+1, pcs[:])
	var st stack = pcs[0:n]
	return st
}

func AsRecoveredPanicError(err error) *RecoveredPanicError {
	if pe, found := unwrapAll(err).(*RecoveredPanicError); found {
		return pe
	}
	return nil
}

type causal interface {
	Cause() error
}

type unwrappable interface {
	Unwrap() error
}

func unwrapOnce(err error) error {
	switch e := err.(type) {
	case causal:
		return e.Cause()
	case unwrappable:
		return e.Unwrap()
	}
	return nil
}

func unwrapAll(err error) error {
	for {
		if cause := unwrapOnce(err); cause != nil {
			err = cause
			continue
		}
		break
	}
	return err
}
