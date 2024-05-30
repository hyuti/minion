package minion

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
)

var ErrTimeout = errors.New("timeout elapsed")

// Minion is basically a function wrapping your passing functions with internal additional logics.
type Minion func()

// Gru running multiple tasks concurrently by taking advantage of goroutines
// A tool to reduce boilerplate of handling goroutines.
type Gru[T any] struct {
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

func (w *Gru[T]) Wrap(ctx context.Context, job func(context.Context) T) Minion {
	return func() {
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
}

func (w *Gru[T]) StartWithCtx(ctx context.Context, jobs ...func(context.Context) T) {
	minions := make([]Minion, len(jobs))
	for idx, job := range jobs {
		minions[idx] = w.Wrap(ctx, job)
	}
	w.wg.Add(len(minions))
	for idx := range minions {
		go func(_idx int) {
			defer w.wg.Done()
			minions[_idx]()
		}(idx)
	}

	if w.eventJob != nil {
		w.wg.Add(1)
		go func() {
			defer w.wg.Done()
			w.eventJob(len(minions))
		}()
	}

	w.wg.Wait()
}

func (w *Gru[T]) Start(jobs ...func() T) {
	ctx := context.Background()
	ctxJobs := make([]func(context.Context) T, len(jobs))
	for idx, job := range jobs {
		ctxJobs[idx] = func(_ context.Context) T {
			return job()
		}
	}
	w.StartWithCtx(ctx, ctxJobs...)
}

// Clean should be called after done using this module
func (w *Gru[T]) Clean() {
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
