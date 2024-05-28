package minion

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
)

var ErrTimeout = errors.New("timeout elapsed")

// Gru running multiple tasks concurrently by taking advantage of goroutines
// A tool to reduce boilerplate of handling goroutines.
type Gru[T any] struct {
	jobs       []func()
	eventJob   func(int)
	errHandler func(error)
	wg         sync.WaitGroup
	ch         chan T
	errCh      chan error
	errs       []error
}

func New[T any]() *Gru[T] {
	w := new(Gru[T])
	return w
}

func (w *Gru[T]) AddMinion(job func() T) *Gru[T] {
	return w.AddMinionWithCtx(context.Background(), func(_ context.Context) T {
		return job()
	})
}

func (w *Gru[T]) AddMinionWithCtx(ctx context.Context, job func(context.Context) T) *Gru[T] {
	wrapper := func() {
		defer func() {
			if r := recover(); r != nil {
				if w.errCh == nil {
					return
				}
				w.errCh <- errors.New(string(debug.Stack()))
			}
		}()
		resultChan := make(chan T)
		go func(c chan T) {
			c <- job(ctx)
			close(resultChan)
		}(resultChan)

		select {
		case <-ctx.Done():
			if w.errCh != nil {
				err := fmt.Errorf("%w: %s", ErrTimeout, ctx.Err())
				w.errCh <- err
			}
		case t := <-resultChan:
			if w.ch != nil {
				w.ch <- t
			}
		}
	}
	w.jobs = append(w.jobs, wrapper)
	return w
}

func (w *Gru[T]) Start() {
	for idx := range w.jobs {
		w.wg.Add(1)
		go func(_idx int) {
			defer w.wg.Done()
			w.jobs[_idx]()
		}(idx)
	}
	if w.eventJob != nil {
		w.wg.Add(1)
		go func() {
			defer w.wg.Done()
			w.eventJob(len(w.jobs))
		}()
	}

	w.wg.Wait()
	w.clean()
}

func (w *Gru[T]) clean() {
	if w.ch != nil {
		close(w.ch)
	}
	if w.errCh != nil {
		close(w.errCh)
	}
}

func (w *Gru[T]) WithCustomErrHandler(h func(error)) {
	w.errCh = make(chan error, 1)
	w.errHandler = h
}

func (w *Gru[T]) WithEvent(h func(T)) {
	if w.errHandler == nil {
		w.WithErrHandler()
	}

	w.ch = make(chan T, 1)
	w.eventJob = func(limit int) {
		defer func() {
			if r := recover(); r != nil {
				if w.errCh == nil {
					return
				}
				w.errCh <- errors.New(string(debug.Stack()))
			}
		}()
		for ; limit > 0; limit -= 1 {
			select {
			case r := <-w.ch:
				h(r)
			case err := <-w.errCh:
				w.errHandler(err)
			}
		}
	}
}

func (w *Gru[T]) WithErrHandler() {
	w.WithCustomErrHandler(func(err error) {
		w.errs = append(w.errs, err)
	})
}
func (w *Gru[T]) Error() error {
	return ErrsToErr(w.errs)
}

func ErrsToErr(errs []error) error {
	return errors.Join(errs...)
}
