// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package expr

import (
	"context"
	"runtime"
	"sync"

	"github.com/Carbonfrost/joe-cli"
	"golang.org/x/sync/errgroup"
)

// Scheduler runs the tasks which evaluators submit using Go.  A scheduler
// can run a task concurrently, in which case it returns before the task
// completes, and the task's error is reported by whatever waits for the
// scheduler (for example, JobsScheduler.Wait).  Otherwise, the task runs inline, and
// its error is returned.
type Scheduler interface {
	Go(ctx context.Context, task func(context.Context) error) error
}

// JobsScheduler provides a scheduler which runs at most a maximum number of tasks
// concurrently.  When that many tasks are already running, a task which is
// submitted runs inline on the goroutine which submitted it, which bounds
// the number of goroutines and allows tasks to submit more tasks without
// deadlock.  The first task that returns an error cancels the context
// provided by NewJobsScheduler.
type JobsScheduler struct {
	group *errgroup.Group
}

type schedulerKey struct{}

type inlineScheduler struct{}

// releasingScheduler invokes release before submitting a task
type releasingScheduler struct {
	Scheduler
	release func()
}

type asyncEvaluator struct {
	inner Evaluator
}

type asyncActionEvaluator struct {
	*asyncEvaluator
	cli.Action
}

// parallelStage serializes a binding evaluator up to the point where it
// first yields or submits a task using Go
type parallelStage struct {
	BindingEvaluator
	mu *sync.Mutex
}

// evaluatorBinder provides an evaluator from values which are bound
// from the context.  Binding is separated from evaluation so that
// Async can bind values before evaluation is scheduled.
type evaluatorBinder func(context.Context) (Evaluator, error)

// exprLocks provides the lock for each *Expr.  All uses of an Expr in an
// expression share the Args of the Expr, so a lock guards the values
// which each use applies to them.
var exprLocks sync.Map

// WithScheduler provides a context which uses the given scheduler for
// the tasks which are submitted using Go
func WithScheduler(ctx context.Context, s Scheduler) context.Context {
	return context.WithValue(ctx, schedulerKey{}, s)
}

// Sync provides a context in which tasks submitted using Go run inline.  An
// evaluator which depends upon whether another evaluator yields a value
// should evaluate it with Sync so that yielding occurs before Evaluate
// returns.  ComposeEvaluator does this for the evaluators it composes.
func Sync(ctx context.Context) context.Context {
	return WithScheduler(ctx, inlineScheduler{})
}

// Go submits a task to the scheduler from the context, which is how an
// evaluator makes work asynchronous.  The task usually ends by yielding its
// result, which continues the expression pipeline:
//
//	func (h hashEval) Evaluate(ctx context.Context, v any, yield func(any) error) error {
//	    return expr.Go(ctx, func(ctx context.Context) error {
//	        sum, err := hashFile(ctx, v.(string))
//	        if err != nil {
//	            return err
//	        }
//	        return yield(sum)
//	    })
//	}
//
// When the context has no scheduler, as with the default compiler, the task
// runs inline, so the evaluator behaves as though it were synchronous.  With
// a scheduler such as the one provided by the Parallel compiler, the task
// can run concurrently and yield after Evaluate has returned.  The values of
// the args of an expression operator must be obtained before calling Go
// because another use of the operator could change them afterwards.
func Go(ctx context.Context, task func(context.Context) error) error {
	return schedulerOf(ctx).Go(ctx, task)
}

// NewJobsScheduler creates a scheduler which runs at most max tasks concurrently.
// When max is zero or negative, runtime.NumCPU() is used.  The context which
// is returned uses the scheduler and is canceled when a task returns an
// error or when Wait returns.
func NewJobsScheduler(ctx context.Context, max int) (*JobsScheduler, context.Context) {
	if max <= 0 {
		max = runtime.NumCPU()
	}
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(max)
	j := &JobsScheduler{group: g}
	return j, WithScheduler(ctx, j)
}

// Go submits a task, which runs concurrently when fewer than the maximum
// number of tasks are running, or otherwise inline.  The error from a task
// which runs inline is returned.
func (j *JobsScheduler) Go(ctx context.Context, task func(context.Context) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if j.group.TryGo(func() error { return task(ctx) }) {
		return nil
	}
	return task(ctx)
}

// Wait waits for all tasks to complete and returns the first error from
// any of them
func (j *JobsScheduler) Wait() error {
	return j.group.Wait()
}

// goWait submits a task, blocking until it can run concurrently.  This must
// only be used from outside of the tasks of j because otherwise, blocking
// can deadlock.
func (j *JobsScheduler) goWait(ctx context.Context, task func(context.Context) error) {
	j.group.Go(func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return task(ctx)
	})
}

// Async provides an evaluator which submits the evaluation of ev as a task
// using Go, which allows a simple, blocking evaluator to be evaluated
// concurrently by the Parallel compiler.  The value of ev is converted
// using the rules of EvaluatorOf.  Using Async asserts that ev is safe to
// evaluate concurrently.
//
// When ev is provided by BindEvaluator, its values are bound before the
// task is submitted.  Otherwise, ev is evaluated entirely within the task,
// so it must not obtain the values of the args of the expression operator
// from the context.
func Async(ev any) Evaluator {
	res := newAsyncEvaluator(EvaluatorOf(ev))
	if a, ok := ev.(cli.Action); ok {
		// Retain the action so that the Expr is initialized, and because
		// initialization can replace the evaluator of the Expr, ensure that
		// it remains asynchronous
		return &asyncActionEvaluator{
			asyncEvaluator: res,
			Action:         cli.Pipeline(a, ensureAsync),
		}
	}
	return res
}

// Parallel provides a compiler whose pipeline evaluates the tasks that
// evaluators submit using Go concurrently, running at most jobs tasks at once.
// When jobs is zero or negative, runtime.NumCPU() is used.  The binding
// evaluators are compiled by next, or by Compile when next is nil.
//
// Evaluate returns after all of the tasks complete, and it returns the first
// error from any of them, which also cancels the context of the others.  If
// the context already has a scheduler other than the one provided by Sync,
// as within Expression.EvaluateParallel, that scheduler is used instead,
// jobs is ignored, and whatever provided the scheduler waits for the tasks.
// Values are yielded to the yielder passed to Evaluate one at a time, but
// their order is not guaranteed.
//
// Unless its evaluator is safe to evaluate concurrently (see Evaluator),
// each binding evaluator is serialized from when it is evaluated until it
// first yields a value, submits a task, or returns.  All uses of an Expr are
// serialized together because they share its args.  Code which runs after
// that point can run concurrently with other evaluations, so it must not
// access shared state without synchronization.
//
// Binding evaluators which next adds to the sequence are not serialized.
// To rewrite the sequence and have the result serialized, apply Parallel to
// the rewritten sequence:
//
//	&expr.Expression{
//	    Compiler: func(items []expr.BindingEvaluator) expr.Evaluator {
//	        return expr.Parallel(8, nil)(append(items, implicitPrint))
//	    },
//	}
func Parallel(jobs int, next Compiler) Compiler {
	if next == nil {
		next = Compile
	}
	return func(items []BindingEvaluator) (Evaluator, error) {
		staged := make([]BindingEvaluator, len(items))
		for i, item := range items {
			staged[i] = newParallelStage(item)
		}
		pipe, err := next(staged)
		if err != nil {
			return nil, err
		}

		var yieldMu sync.Mutex
		return evaluatorFunc(func(ctx context.Context, v any, yield func(any) error) error {
			if yield == nil {
				yield = emptyYielder
			}
			serialYield := func(v any) error {
				yieldMu.Lock()
				defer yieldMu.Unlock()
				return yield(v)
			}

			if hasScheduler(ctx) {
				return pipe.Evaluate(ctx, v, serialYield)
			}

			j, ctx := NewJobsScheduler(ctx, jobs)
			j.goWait(ctx, func(ctx context.Context) error {
				return pipe.Evaluate(ctx, v, serialYield)
			})
			return j.Wait()
		}), nil
	}
}

func (inlineScheduler) Go(ctx context.Context, task func(context.Context) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return task(ctx)
}

func (r releasingScheduler) Go(ctx context.Context, task func(context.Context) error) error {
	r.release()
	return r.Scheduler.Go(ctx, task)
}

func newAsyncEvaluator(ev Evaluator) *asyncEvaluator {
	if e, ok := ev.(*evaluatorInit); ok {
		// Unwrap to detect whether the evaluator is an evaluatorBinder
		ev = e.Evaluator
	}
	return &asyncEvaluator{inner: ev}
}

func (a *asyncEvaluator) Evaluate(ctx context.Context, v any, yield func(any) error) error {
	ev := a.inner
	if b, ok := ev.(evaluatorBinder); ok {
		var err error
		if ev, err = b.bind(ctx); err != nil {
			return err
		}
	}
	return Go(ctx, func(ctx context.Context) error {
		return ev.Evaluate(ctx, v, yield)
	})
}

// Concurrent indicates that the evaluator is safe to evaluate concurrently
func (*asyncEvaluator) Concurrent() bool {
	return true
}

func (b evaluatorBinder) bind(ctx context.Context) (Evaluator, error) {
	return b(ctx)
}

func (b evaluatorBinder) Evaluate(ctx context.Context, v any, yield func(any) error) error {
	ev, err := b(ctx)
	if err != nil {
		return err
	}
	return ev.Evaluate(ctx, v, yield)
}

func newParallelStage(item BindingEvaluator) BindingEvaluator {
	var mu *sync.Mutex
	if e, ok := item.(*boundExpr); ok {
		if len(e.expr.Args) > 0 || !isConcurrent(e.expr.Evaluate) {
			l, _ := exprLocks.LoadOrStore(e.expr, new(sync.Mutex))
			mu = l.(*sync.Mutex)
		}
	} else if !isConcurrent(item) {
		mu = new(sync.Mutex)
	}
	if mu == nil {
		return item
	}
	return &parallelStage{BindingEvaluator: item, mu: mu}
}

func (s *parallelStage) Evaluate(ctx context.Context, v any, yield func(any) error) error {
	s.mu.Lock()
	release := sync.OnceFunc(s.mu.Unlock)
	defer release()

	ctx = WithScheduler(ctx, releasingScheduler{Scheduler: schedulerOf(ctx), release: release})
	return s.BindingEvaluator.Evaluate(ctx, v, func(v any) error {
		release()
		return yield(v)
	})
}

// isConcurrent detects whether the evaluator is safe to evaluate concurrently
// by the convention of a method Concurrent() bool.  (For exprBinding, which
// embeds the Evaluator interface, the method isn't promoted.)
func isConcurrent(v any) bool {
	if b, ok := v.(*exprBinding); ok {
		v = b.Evaluator
	}
	c, ok := v.(interface{ Concurrent() bool })
	return ok && c.Concurrent()
}

func ensureAsync(c *cli.Context) error {
	if e, ok := c.Target().(*Expr); ok {
		var isAsync bool
		switch e.Evaluate.(type) {
		case *asyncEvaluator, *asyncActionEvaluator:
			isAsync = true
		}

		if !isAsync {
			e.Evaluate = newAsyncEvaluator(EvaluatorOf(e.Evaluate))
		}
	}
	return nil
}

func schedulerOf(ctx context.Context) Scheduler {
	if s, ok := ctx.Value(schedulerKey{}).(Scheduler); ok && s != nil {
		return s
	}
	return inlineScheduler{}
}

func hasScheduler(ctx context.Context) bool {
	_, inline := schedulerOf(ctx).(inlineScheduler)
	return !inline
}

// cliContextOf obtains the *cli.Context from ctx.  When ctx wraps the
// *cli.Context, the result is a copy whose context values are provided by
// ctx, which retains values such as the scheduler and cancellation.
func cliContextOf(ctx context.Context) *cli.Context {
	c := cli.FromContext(ctx)
	if context.Context(c) == ctx {
		return c
	}

	var res *cli.Context
	_ = cli.WithContext(func(context.Context) context.Context {
		return ctx
	}).ExecuteWithNext(c, cli.ActionFunc(func(wrapped *cli.Context) error {
		res = wrapped
		return nil
	}))
	return res
}

var (
	_ Scheduler       = (*JobsScheduler)(nil)
	_ Evaluator       = (*asyncEvaluator)(nil)
	_ ActionEvaluator = (*asyncActionEvaluator)(nil)
	_ Evaluator       = evaluatorBinder(nil)
)
