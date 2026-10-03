// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package expr_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
	"github.com/Carbonfrost/joe-cli/extensions/expr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Go", func() {

	It("runs the task inline when the context has no scheduler", func() {
		var ran bool
		err := expr.Go(context.Background(), func(context.Context) error {
			ran = true
			return nil
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(ran).To(BeTrue())
	})

	It("returns the error from the task", func() {
		err := expr.Go(context.Background(), func(context.Context) error {
			return errors.New("boom")
		})
		Expect(err).To(MatchError("boom"))
	})

	It("does not run the task when the context is canceled", func() {
		var ran bool
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := expr.Go(ctx, func(context.Context) error {
			ran = true
			return nil
		})

		Expect(err).To(MatchError(context.Canceled))
		Expect(ran).To(BeFalse())
	})

	It("runs the task inline within Sync", func() {
		j, ctx := expr.NewJobsScheduler(context.Background(), 2)
		var ran bool
		err := expr.Go(expr.Sync(ctx), func(context.Context) error {
			ran = true
			return nil
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(ran).To(BeTrue())
		Expect(j.Wait()).To(Succeed())
	})
})

var _ = Describe("NewJobsScheduler", func() {

	It("runs tasks concurrently up to the maximum and then inline", func() {
		j, ctx := expr.NewJobsScheduler(context.Background(), 2)
		b := newBarrier(2)
		release := make(chan struct{})

		for range 2 {
			Expect(j.Go(ctx, func(context.Context) error {
				if err := b.wait(); err != nil {
					return err
				}
				<-release
				return nil
			})).To(Succeed())
		}
		Expect(b.wait()).To(Succeed())

		var inline bool
		Expect(j.Go(ctx, func(context.Context) error {
			inline = true
			return nil
		})).To(Succeed())
		Expect(inline).To(BeTrue())

		close(release)
		Expect(j.Wait()).To(Succeed())
	})

	It("returns the error from a task which runs inline", func() {
		j, ctx := expr.NewJobsScheduler(context.Background(), 1)
		release := make(chan struct{})
		Expect(j.Go(ctx, func(context.Context) error {
			<-release
			return nil
		})).To(Succeed())

		err := j.Go(ctx, func(context.Context) error {
			return errors.New("boom")
		})
		close(release)

		Expect(err).To(MatchError("boom"))
		Expect(j.Wait()).To(Succeed())
	})

	It("returns the first error from Wait and cancels the context", func() {
		j, ctx := expr.NewJobsScheduler(context.Background(), 2)
		_ = j.Go(ctx, func(context.Context) error {
			return errors.New("boom")
		})

		Expect(j.Wait()).To(MatchError("boom"))
		Expect(ctx.Err()).To(HaveOccurred())
	})
})

var _ = Describe("Async", func() {

	It("evaluates inline without a scheduler", func() {
		var yielded any
		ev := expr.Async(expr.NewEvaluator(func(_ context.Context, v string, yield func(any) error) error {
			return yield(v + "!")
		}))

		err := ev.Evaluate(context.Background(), "item", func(v any) error {
			yielded = v
			return nil
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(yielded).To(Equal("item!"))
	})

	It("is safe to evaluate concurrently", func() {
		ev := expr.Async(expr.AlwaysTrue)
		Expect(ev.(interface{ Concurrent() bool }).Concurrent()).To(BeTrue())
	})

	It("binds values before evaluation and remains asynchronous after initialization", func() {
		var (
			bound      string
			concurrent bool
		)
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Name: "start",
					NArg: cli.TakeUntilNextFlag,
				},
				{
					Name: "e",
					Value: &expr.Expression{
						Exprs: []*expr.Expr{
							{
								Name: "tag",
								Args: cli.Args("value", new(string)),
								Evaluate: expr.Async(expr.BindEvaluator(func(s string) expr.Evaluator {
									return expr.EvaluatorOf(func() { bound = s })
								}, bind.String())),
							},
						},
					},
				},
			},
			Action: func(c context.Context) error {
				e := expr.FromContext(c, "e")
				_, concurrent = e.Exprs[0].Evaluate.(interface{ Concurrent() bool })
				return e.Evaluate(c, "item")
			},
		}
		args, _ := cli.Split("app _ -tag a")
		err := app.RunContext(context.Background(), args...)

		Expect(err).NotTo(HaveOccurred())
		Expect(bound).To(Equal("a"))
		Expect(concurrent).To(BeTrue())
	})
})

var _ = Describe("Parallel", func() {

	var (
		fanOut = func(n int) expr.BindingEvaluator {
			return expr.NewBindingEvaluator(expr.EvaluatorOf(func(_ any, yield func(any) error) error {
				for i := range n {
					if err := yield(i); err != nil {
						return err
					}
				}
				return nil
			}))
		}
		async = func(fn func(v any) error) expr.BindingEvaluator {
			return expr.NewBindingEvaluator(expr.Async(func(v any, yield func(any) error) error {
				if err := fn(v); err != nil {
					return err
				}
				return yield(v)
			}))
		}
		collect = func(results *[]any) func(any) error {
			return func(v any) error {
				*results = append(*results, v)
				return nil
			}
		}
		tracker = func(active, maxActive *atomic.Int32) func(any) error {
			return func(any) error {
				n := active.Add(1)
				defer active.Add(-1)
				for {
					m := maxActive.Load()
					if n <= m || maxActive.CompareAndSwap(m, n) {
						break
					}
				}
				time.Sleep(time.Millisecond)
				return nil
			}
		}
	)

	It("evaluates tasks concurrently and waits for them", func() {
		var results []any
		b := newBarrier(3)
		pipe, _ := expr.Parallel(4, nil)([]expr.BindingEvaluator{
			fanOut(3),
			async(func(any) error { return b.wait() }),
		})

		err := pipe.Evaluate(context.Background(), nil, collect(&results))
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(ConsistOf(0, 1, 2))
	})

	It("limits the number of concurrent tasks", func() {
		var (
			results           []any
			active, maxActive atomic.Int32
		)
		pipe, _ := expr.Parallel(3, nil)([]expr.BindingEvaluator{
			fanOut(30),
			async(tracker(&active, &maxActive)),
		})

		err := pipe.Evaluate(context.Background(), nil, collect(&results))
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(30))
		Expect(maxActive.Load()).To(BeNumerically("<=", 3))
	})

	It("returns the first error from a task", func() {
		pipe, _ := expr.Parallel(4, nil)([]expr.BindingEvaluator{
			fanOut(5),
			async(func(v any) error {
				if v == 2 {
					return errors.New("boom")
				}
				return nil
			}),
		})

		err := pipe.Evaluate(context.Background(), nil, nil)
		Expect(err).To(MatchError("boom"))
	})

	It("serializes evaluators which aren't safe to evaluate concurrently", func() {
		var count int
		pipe, _ := expr.Parallel(8, nil)([]expr.BindingEvaluator{
			fanOut(50),
			async(func(any) error { return nil }),
			expr.NewBindingEvaluator(expr.EvaluatorOf(func(any) { count++ })),
		})

		err := pipe.Evaluate(context.Background(), nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(count).To(Equal(50))
	})

	It("does not serialize evaluators which are safe to evaluate concurrently", func() {
		pipe, _ := expr.Parallel(4, nil)([]expr.BindingEvaluator{
			fanOut(2),
			async(func(any) error { return nil }),
			expr.NewBindingEvaluator(concurrentEvaluator(newBarrier(2).wait)),
		})

		err := pipe.Evaluate(context.Background(), nil, nil)
		Expect(err).NotTo(HaveOccurred())
	})

	It("evaluates the evaluators of ComposeEvaluator synchronously", func() {
		var calls atomic.Int32
		pipe, _ := expr.Parallel(4, nil)([]expr.BindingEvaluator{
			expr.NewBindingEvaluator(expr.ComposeEvaluator(
				expr.Async(expr.AlwaysTrue),
				expr.EvaluatorOf(func() bool { calls.Add(1); return true }),
			)),
		})

		err := pipe.Evaluate(context.Background(), "item", nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(calls.Load()).To(BeZero())
	})

	It("uses the scheduler from EvaluateParallel", func() {
		var active, maxActive atomic.Int32
		e := &expr.Expression{Compiler: expr.Parallel(100, nil)}
		e.Append(async(tracker(&active, &maxActive)))

		items := make([]any, 30)
		err := e.EvaluateParallel(context.Background(), 2, items...)
		Expect(err).NotTo(HaveOccurred())
		Expect(maxActive.Load()).To(BeNumerically("<=", 2))
	})

	It("evaluates the tasks of expression operators concurrently", func() {
		b := newBarrier(3)
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Name: "start",
					NArg: cli.TakeUntilNextFlag,
				},
				{
					Name: "e",
					Value: &expr.Expression{
						Compiler: expr.Parallel(4, nil),
						Exprs: []*expr.Expr{
							{
								Name: "fan",
								Evaluate: func(_ any, yield func(any) error) error {
									for i := range 3 {
										if err := yield(i); err != nil {
											return err
										}
									}
									return nil
								},
							},
							{
								Name:     "wait",
								Evaluate: expr.Async(func(any) error { return b.wait() }),
							},
						},
					},
				},
			},
			Action: func(c context.Context) error {
				return expr.FromContext(c, "e").Evaluate(c, nil)
			},
		}
		args, _ := cli.Split("app _ -fan -wait")
		err := app.RunContext(context.Background(), args...)

		Expect(err).NotTo(HaveOccurred())
	})

	It("evaluates each use of an expression operator with its own args", func() {
		var (
			mu      sync.Mutex
			results []any
		)
		collector := expr.NewBindingEvaluator(expr.EvaluatorOf(func(v any) {
			mu.Lock()
			defer mu.Unlock()
			results = append(results, v)
		}))
		tag := expr.Async(expr.BindEvaluator(func(s string) expr.Evaluator {
			return expr.EvaluatorOf(func(_ context.Context, v any, yield func(any) error) error {
				time.Sleep(time.Millisecond)
				return yield(fmt.Sprintf("%v/%s", v, s))
			})
		}, bind.String()))

		app := &cli.App{
			Args: []*cli.Arg{
				{
					Name: "start",
					NArg: cli.TakeUntilNextFlag,
				},
				{
					Name: "e",
					Value: &expr.Expression{
						Compiler: func(items []expr.BindingEvaluator) (expr.Evaluator, error) {
							return expr.Parallel(4, nil)(append(items, collector))
						},
						Exprs: []*expr.Expr{
							{
								Name:     "tag",
								Args:     cli.Args("value", new(string)),
								Evaluate: tag,
							},
						},
					},
				},
			},
			Action: func(c context.Context) error {
				return expr.FromContext(c, "e").EvaluateParallel(c, 4, 1, 2, 3, 4, 5, 6, 7, 8)
			},
		}
		args, _ := cli.Split("app _ -tag a -tag b")
		err := app.RunContext(context.Background(), args...)

		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(ConsistOf(
			"1/a/b", "2/a/b", "3/a/b", "4/a/b", "5/a/b", "6/a/b", "7/a/b", "8/a/b",
		))
	})
})

type concurrentEvaluator func() error

func (c concurrentEvaluator) Evaluate(_ context.Context, v any, yield func(any) error) error {
	if err := c(); err != nil {
		return err
	}
	return yield(v)
}

func (concurrentEvaluator) Concurrent() bool {
	return true
}

// barrier waits until n participants have arrived, which demonstrates that
// they run concurrently
type barrier struct {
	n     int32
	count atomic.Int32
	done  chan struct{}
}

func newBarrier(n int32) *barrier {
	return &barrier{n: n, done: make(chan struct{})}
}

func (b *barrier) wait() error {
	if b.count.Add(1) == b.n {
		close(b.done)
	}
	select {
	case <-b.done:
		return nil
	case <-time.After(5 * time.Second):
		return errors.New("timed out waiting for concurrent evaluations")
	}
}
