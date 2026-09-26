// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package operator_test

import (
	"context"
	"errors"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/expr"
	"github.com/Carbonfrost/joe-cli/extensions/expr/operator"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Operators", func() {

	Describe("Not", func() {

		It("yields when wrapped evaluator does not yield", func() {
			var yielded bool
			notEval := operator.Not(expr.AlwaysFalse)

			err := notEval.Evaluate(context.Background(), "value", func(v any) error {
				yielded = true
				Expect(v).To(Equal("value"))
				return nil
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(yielded).To(BeTrue())
		})

		It("does not yield when wrapped evaluator yields", func() {
			var yielded bool
			notEval := operator.Not(expr.AlwaysTrue)

			err := notEval.Evaluate(context.Background(), "value", func(v any) error {
				yielded = true
				return nil
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(yielded).To(BeFalse())
		})

		It("inverts predicate behavior", func() {
			isEven := expr.Predicate(func(v any) bool {
				return v.(int)%2 == 0
			})
			notEven := operator.Not(isEven)

			var results []int
			yield := func(v any) error {
				results = append(results, v.(int))
				return nil
			}

			// Even number - should not yield (inverted)
			err := notEven.Evaluate(context.Background(), 2, yield)
			Expect(err).NotTo(HaveOccurred())

			// Odd number - should yield (inverted)
			err = notEven.Evaluate(context.Background(), 3, yield)
			Expect(err).NotTo(HaveOccurred())

			Expect(results).To(Equal([]int{3}))
		})

		It("propagates errors from wrapped evaluator", func() {
			expectedErr := errors.New("evaluation failed")
			failingEval := expr.EvaluatorOf(func() error {
				return expectedErr
			})
			notEval := operator.Not(failingEval)

			err := notEval.Evaluate(context.Background(), "value", func(v any) error {
				return nil
			})

			Expect(err).To(MatchError(expectedErr))
		})
	})

	Describe("Or", func() {

		It("yields if left evaluator yields (short-circuit)", func() {
			var leftCalled, rightCalled bool
			left := expr.EvaluatorOf(func(c context.Context, v any, y func(any) error) error {
				leftCalled = true
				return y(v) // yields
			})
			right := expr.EvaluatorOf(func(c context.Context, v any, y func(any) error) error {
				rightCalled = true
				return y(v)
			})

			orEval := operator.Or(left, right)
			var yielded bool
			err := orEval.Evaluate(context.Background(), "value", func(v any) error {
				yielded = true
				return nil
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(yielded).To(BeTrue())
			Expect(leftCalled).To(BeTrue())
			Expect(rightCalled).To(BeFalse(), "right should not be evaluated due to short-circuit")
		})

		It("evaluates right if left does not yield", func() {
			var leftCalled, rightCalled bool
			left := expr.EvaluatorOf(func(c context.Context, v any, y func(any) error) error {
				leftCalled = true
				return nil // does not yield
			})
			right := expr.EvaluatorOf(func(c context.Context, v any, y func(any) error) error {
				rightCalled = true
				return y(v) // yields
			})

			orEval := operator.Or(left, right)
			var yielded bool
			err := orEval.Evaluate(context.Background(), "value", func(v any) error {
				yielded = true
				return nil
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(yielded).To(BeTrue())
			Expect(leftCalled).To(BeTrue())
			Expect(rightCalled).To(BeTrue())
		})

		It("does not yield if neither side yields", func() {
			orEval := operator.Or(expr.AlwaysFalse, expr.AlwaysFalse)
			var yielded bool

			err := orEval.Evaluate(context.Background(), "value", func(v any) error {
				yielded = true
				return nil
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(yielded).To(BeFalse())
		})

		It("works with predicates", func() {
			isEven := expr.Predicate(func(v any) bool {
				return v.(int)%2 == 0
			})
			isNegative := expr.Predicate(func(v any) bool {
				return v.(int) < 0
			})

			orEval := operator.Or(isEven, isNegative)
			var results []int
			yield := func(v any) error {
				results = append(results, v.(int))
				return nil
			}

			// Test even positive (should yield via left)
			err := orEval.Evaluate(context.Background(), 2, yield)
			Expect(err).NotTo(HaveOccurred())

			// Test odd negative (should yield via right)
			err = orEval.Evaluate(context.Background(), -3, yield)
			Expect(err).NotTo(HaveOccurred())

			// Test odd positive (should not yield)
			err = orEval.Evaluate(context.Background(), 3, yield)
			Expect(err).NotTo(HaveOccurred())

			Expect(results).To(Equal([]int{2, -3}))
		})

		It("propagates errors from left evaluator", func() {
			expectedErr := errors.New("left failed")
			left := expr.EvaluatorOf(func() error {
				return expectedErr
			})
			orEval := operator.Or(left, expr.AlwaysTrue)

			err := orEval.Evaluate(context.Background(), "value", func(v any) error {
				return nil
			})

			Expect(err).To(MatchError(expectedErr))
		})

		It("propagates errors from right evaluator", func() {
			expectedErr := errors.New("right failed")
			right := expr.EvaluatorOf(func() error {
				return expectedErr
			})
			orEval := operator.Or(expr.AlwaysFalse, right)

			err := orEval.Evaluate(context.Background(), "value", func(v any) error {
				return nil
			})

			Expect(err).To(MatchError(expectedErr))
		})

		It("generates expected expr from Prototype", func() {
			e := &expr.Expr{Evaluate: operator.Or(nil, nil)}
			app := &cli.App{
				Args: []*cli.Arg{
					{
						Name: "e",
						Value: &expr.Expression{
							Exprs: []*expr.Expr{e},
						},
					},
				},
			}

			app.Initialize(context.Background())

			Expect(e.HelpText).To(Equal("The logical OR operator"))
			Expect(e.Name).To(Equal("or"))
			Expect(e.Aliases).To(Equal([]string{"o"}))
		})
	})

	Describe("And", func() {

		It("yields if both evaluators yield", func() {
			andEval := operator.And(expr.AlwaysTrue, expr.AlwaysTrue)
			var yielded bool

			err := andEval.Evaluate(context.Background(), "value", func(v any) error {
				yielded = true
				return nil
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(yielded).To(BeTrue())
		})

		It("does not yield if left evaluator does not yield (short-circuit)", func() {
			var leftCalled, rightCalled bool
			left := expr.EvaluatorOf(func(c context.Context, v any, y func(any) error) error {
				leftCalled = true
				return nil // does not yield
			})
			right := expr.EvaluatorOf(func(c context.Context, v any, y func(any) error) error {
				rightCalled = true
				return y(v)
			})

			andEval := operator.And(left, right)
			var yielded bool
			err := andEval.Evaluate(context.Background(), "value", func(v any) error {
				yielded = true
				return nil
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(yielded).To(BeFalse())
			Expect(leftCalled).To(BeTrue())
			Expect(rightCalled).To(BeFalse(), "right should not be evaluated due to short-circuit")
		})

		It("does not yield if right evaluator does not yield", func() {
			andEval := operator.And(expr.AlwaysTrue, expr.AlwaysFalse)
			var yielded bool

			err := andEval.Evaluate(context.Background(), "value", func(v any) error {
				yielded = true
				return nil
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(yielded).To(BeFalse())
		})

		It("works with predicates", func() {
			isEven := expr.Predicate(func(v any) bool {
				return v.(int)%2 == 0
			})
			isPositive := expr.Predicate(func(v any) bool {
				return v.(int) > 0
			})

			andEval := operator.And(isEven, isPositive)
			var results []int
			yield := func(v any) error {
				results = append(results, v.(int))
				return nil
			}

			// Test even positive (should yield)
			err := andEval.Evaluate(context.Background(), 2, yield)
			Expect(err).NotTo(HaveOccurred())

			// Test even negative (should not yield)
			err = andEval.Evaluate(context.Background(), -2, yield)
			Expect(err).NotTo(HaveOccurred())

			// Test odd positive (should not yield)
			err = andEval.Evaluate(context.Background(), 3, yield)
			Expect(err).NotTo(HaveOccurred())

			Expect(results).To(Equal([]int{2}))
		})

		It("propagates errors from left evaluator", func() {
			expectedErr := errors.New("left failed")
			left := expr.EvaluatorOf(func() error {
				return expectedErr
			})
			andEval := operator.And(left, expr.AlwaysTrue)

			err := andEval.Evaluate(context.Background(), "value", func(v any) error {
				return nil
			})

			Expect(err).To(MatchError(expectedErr))
		})

		It("propagates errors from right evaluator", func() {
			expectedErr := errors.New("right failed")
			right := expr.EvaluatorOf(func() error {
				return expectedErr
			})
			andEval := operator.And(expr.AlwaysTrue, right)

			err := andEval.Evaluate(context.Background(), "value", func(v any) error {
				return nil
			})

			Expect(err).To(MatchError(expectedErr))
		})

		It("generates expected expr from Prototype", func() {
			e := &expr.Expr{Evaluate: operator.And(nil, nil)}
			app := &cli.App{
				Args: []*cli.Arg{
					{
						Name: "e",
						Value: &expr.Expression{
							Exprs: []*expr.Expr{e},
						},
					},
				},
			}

			app.Initialize(context.Background())

			Expect(e.HelpText).To(Equal("The logical AND operator"))
			Expect(e.Name).To(Equal("and"))
			Expect(e.Aliases).To(Equal([]string{"a"}))
		})
	})

	Describe("Complex combinations", func() {

		It("combines NOT and OR", func() {
			// !(even OR negative) should yield odd positives only
			isEven := expr.Predicate(func(v any) bool {
				return v.(int)%2 == 0
			})
			isNegative := expr.Predicate(func(v any) bool {
				return v.(int) < 0
			})

			combined := operator.Not(operator.Or(isEven, isNegative))
			var results []int
			yield := func(v any) error {
				results = append(results, v.(int))
				return nil
			}

			testValues := []int{-4, -3, -2, -1, 0, 1, 2, 3, 4}
			for _, val := range testValues {
				err := combined.Evaluate(context.Background(), val, yield)
				Expect(err).NotTo(HaveOccurred())
			}

			// Should only include odd positives: 1, 3
			Expect(results).To(Equal([]int{1, 3}))
		})

		It("combines NOT and AND", func() {
			// !(even AND positive) should yield odds and negatives
			isEven := expr.Predicate(func(v any) bool {
				return v.(int)%2 == 0
			})
			isPositive := expr.Predicate(func(v any) bool {
				return v.(int) > 0
			})

			combined := operator.Not(operator.And(isEven, isPositive))
			var results []int
			yield := func(v any) error {
				results = append(results, v.(int))
				return nil
			}

			testValues := []int{-2, -1, 0, 1, 2, 3, 4}
			for _, val := range testValues {
				err := combined.Evaluate(context.Background(), val, yield)
				Expect(err).NotTo(HaveOccurred())
			}

			// Should exclude even positives (2, 4), so: -2, -1, 0, 1, 3
			Expect(results).To(Equal([]int{-2, -1, 0, 1, 3}))
		})

		It("combines AND and OR (De Morgan's law)", func() {
			// (even AND positive) OR (odd AND negative)
			isEven := expr.Predicate(func(v any) bool {
				return v.(int)%2 == 0
			})
			isOdd := expr.Predicate(func(v any) bool {
				return v.(int)%2 != 0
			})
			isPositive := expr.Predicate(func(v any) bool {
				return v.(int) > 0
			})
			isNegative := expr.Predicate(func(v any) bool {
				return v.(int) < 0
			})

			combined := operator.Or(
				operator.And(isEven, isPositive),
				operator.And(isOdd, isNegative),
			)
			var results []int
			yield := func(v any) error {
				results = append(results, v.(int))
				return nil
			}

			testValues := []int{-4, -3, -2, -1, 0, 1, 2, 3, 4}
			for _, val := range testValues {
				err := combined.Evaluate(context.Background(), val, yield)
				Expect(err).NotTo(HaveOccurred())
			}

			// Should include even positives (2, 4) and odd negatives (-3, -1)
			Expect(results).To(Equal([]int{-3, -1, 2, 4}))
		})
	})
})
