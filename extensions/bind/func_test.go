// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bind_test

import (
	"context"
	"math/big"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("Action", func() {

	Describe("binding arguments", func() {
		DescribeTable("examples", func(binder bind.Binder[string], expected string) {
			var (
				action     = new(joeclifakes.FakeAction)
				calledWith string
			)
			factory := func(c string) cli.Action {
				calledWith = c
				return action
			}

			app := &cli.App{
				Flags: []*cli.Flag{
					{Name: "flag"},
				},
				Args: []*cli.Arg{
					{Name: "arg"},
				},
				Uses: bind.Action(factory, binder),
			}

			args, _ := cli.Split("app arg_value --flag flag_value")
			err := app.RunContext(context.Background(), args)
			Expect(err).NotTo(HaveOccurred())
			Expect(action.ExecuteCallCount()).To(Equal(1))
			Expect(calledWith).To(Equal(expected))
		},
			Entry("named", bind.String("flag"), "flag_value"),
			Entry("implicit name", bind.String(), "arg_value"),
		)
	})

	It("binds implicitly on the current flag", func() {
		var (
			action     = new(joeclifakes.FakeAction)
			calledWith *big.Int
		)
		factory := func(c *big.Int) cli.Action {
			calledWith = c
			return action
		}

		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name: "flag",
					Uses: bind.Action(factory, bind.BigInt()),
				},
			},
		}

		args, _ := cli.Split("app --flag 8000")
		err := app.RunContext(context.Background(), args)
		Expect(err).NotTo(HaveOccurred())
		Expect(action.ExecuteCallCount()).To(Equal(1))
		Expect(calledWith.Int64()).To(Equal(int64(8000)))
	})

	It("can bind implicit names in Action pipeline", func() {
		var (
			action     = new(joeclifakes.FakeAction)
			calledWith string
		)
		factory := func(c string) cli.Action {
			calledWith = c
			return action
		}

		app := &cli.App{
			Args: []*cli.Arg{
				{Name: "arg"},
			},
			Action: bind.Action(factory, bind.String()),
		}

		args, _ := cli.Split("app arg_value")
		err := app.RunContext(context.Background(), args)
		Expect(err).NotTo(HaveOccurred())
		Expect(action.ExecuteCallCount()).To(Equal(1))
		Expect(calledWith).To(Equal("arg_value"))
	})
})

var _ = Describe("Action2", func() {

	Describe("binding arguments", func() {
		DescribeTable("examples", func(binder1, binder2 bind.Binder[string], expected []string) {
			var (
				action     = new(joeclifakes.FakeAction)
				calledWith []string
			)
			factory := func(a, b string) cli.Action {
				calledWith = []string{a, b}
				return action
			}

			app := &cli.App{
				Flags: []*cli.Flag{
					{Name: "flag"},
				},
				Args: []*cli.Arg{
					{Name: "arg1", NArg: 1},
					{Name: "arg2", NArg: 1},
				},
				Uses: bind.Action2(factory, binder1, binder2),
			}

			args, _ := cli.Split("app arg1_value arg2_value --flag flag_value")
			err := app.RunContext(context.Background(), args)
			Expect(err).NotTo(HaveOccurred())
			Expect(action.ExecuteCallCount()).To(Equal(1))
			Expect(calledWith).To(Equal(expected))
		},
			Entry("named", bind.String("flag"), bind.String("flag"), []string{"flag_value", "flag_value"}),
			Entry("implicit name", bind.String(), bind.String(), []string{"arg1_value", "arg2_value"}),
		)

	})
})

var _ = Describe("Action3", func() {

	Describe("binding arguments", func() {
		DescribeTable("examples", func(binder1, binder2, binder3 bind.Binder[string], expected []string) {
			var (
				action     = new(joeclifakes.FakeAction)
				calledWith []string
			)
			factory := func(a, b, c string) cli.Action {
				calledWith = []string{a, b, c}
				return action
			}

			app := &cli.App{
				Flags: []*cli.Flag{
					{Name: "flag"},
				},
				Args: []*cli.Arg{
					{Name: "arg1", NArg: 1},
					{Name: "arg2", NArg: 1},
					{Name: "arg3", NArg: 1},
				},
				Uses: bind.Action3(factory, binder1, binder2, binder3),
			}

			args, _ := cli.Split("app arg1_value arg2_value arg3_value --flag flag_value")
			err := app.RunContext(context.Background(), args)
			Expect(err).NotTo(HaveOccurred())
			Expect(action.ExecuteCallCount()).To(Equal(1))
			Expect(calledWith).To(Equal(expected))
		},
			Entry("named", bind.String("flag"), bind.String("flag"), bind.String("flag"), []string{"flag_value", "flag_value", "flag_value"}),
			Entry("implicit name", bind.String(), bind.String(), bind.String(), []string{"arg1_value", "arg2_value", "arg3_value"}),
		)

	})
})

var _ = Describe("Call", func() {

	It("invokes the factory", func() {
		var (
			called     bool
			calledWith string
		)
		factory := func(c string) error {
			called = true
			calledWith = c
			return nil
		}

		actual := bind.Call(factory, bind.String("flag"))
		app := &cli.App{
			Flags: []*cli.Flag{
				{Name: "flag"},
			},
			Action: actual,
		}

		args, _ := cli.Split("app --flag=has_value")
		err := app.RunContext(context.Background(), args)

		Expect(err).NotTo(HaveOccurred())
		Expect(called).To(BeTrue())
		Expect(calledWith).To(Equal("has_value"))
	})
})

var _ = Describe("Call2", func() {

	Describe("binding arguments", func() {
		DescribeTable("examples", func(binder1, binder2 bind.Binder[string], expected []string) {
			var (
				calledWith []string
			)
			factory := func(a, b string) error {
				calledWith = []string{a, b}
				return nil
			}

			app := &cli.App{
				Flags: []*cli.Flag{
					{Name: "flag"},
				},
				Args: []*cli.Arg{
					{Name: "arg1", NArg: 1},
					{Name: "arg2", NArg: 1},
				},
				Uses: bind.Call2(factory, binder1, binder2),
			}

			args, _ := cli.Split("app arg1_value arg2_value --flag flag_value")
			err := app.RunContext(context.Background(), args)
			Expect(err).NotTo(HaveOccurred())
			Expect(calledWith).To(Equal(expected))
		},
			Entry("named", bind.String("flag"), bind.String("flag"), []string{"flag_value", "flag_value"}),
			Entry("implicit name", bind.String(), bind.String(), []string{"arg1_value", "arg2_value"}),
		)
	})
})

var _ = Describe("Call3", func() {

	Describe("binding arguments", func() {
		DescribeTable("examples", func(binder1, binder2, binder3 bind.Binder[string], expected []string) {
			var (
				calledWith []string
			)
			factory := func(a, b, c string) error {
				calledWith = []string{a, b, c}
				return nil
			}

			app := &cli.App{
				Flags: []*cli.Flag{
					{Name: "flag"},
				},
				Args: []*cli.Arg{
					{Name: "arg1", NArg: 1},
					{Name: "arg2", NArg: 1},
					{Name: "arg3", NArg: 1},
				},
				Uses: bind.Call3(factory, binder1, binder2, binder3),
			}

			args, _ := cli.Split("app arg1_value arg2_value arg3_value --flag flag_value")
			err := app.RunContext(context.Background(), args)
			Expect(err).NotTo(HaveOccurred())
			Expect(calledWith).To(Equal(expected))
		},
			Entry("named", bind.String("flag"), bind.String("flag"), bind.String("flag"), []string{"flag_value", "flag_value", "flag_value"}),
			Entry("implicit name", bind.String(), bind.String(), bind.String(), []string{"arg1_value", "arg2_value", "arg3_value"}),
		)
	})
})

var _ = Describe("SetPointer", func() {

	Describe("binding arguments", func() {
		DescribeTable("examples", func(binder bind.Binder[string], expected string) {
			var target string

			app := &cli.App{
				Args: []*cli.Arg{
					{Name: "arg", NArg: 1},
				},
				Flags: []*cli.Flag{
					{Name: "f"},
				},
				Uses: bind.SetPointer(&target, binder),
			}

			args, _ := cli.Split("app -f flag_value arg_value")
			err := app.RunContext(context.Background(), args)
			Expect(err).NotTo(HaveOccurred())
			Expect(target).To(Equal(expected))
		},
			Entry("named", bind.String("f"), "flag_value"),
			Entry("implicit name", bind.String(), "arg_value"),
		)
	})
})

var _ = Describe("Indirect", func() {

	It("copies the implied value of the function", func() {
		fs := &cli.FileSet{Recursive: true}
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name: "no-recursive",
					Uses: bind.Indirect("files", (*cli.FileSet).SetRecursive, false),
				},
			},
			Args: []*cli.Arg{
				{
					Name:  "files",
					Value: fs,
				},
			},
		}
		app.Initialize(context.Background())
		Expect(app.Flags[0].Value).To(Equal(new(bool)))
	})

	It("invokes bind func with static value", func() {
		fs := &cli.FileSet{Recursive: true}
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name:  "no-recursive",
					Value: new(bool),
					Uses:  bind.Indirect("files", (*cli.FileSet).SetRecursive, false),
				},
			},
			Args: []*cli.Arg{
				{
					Name:  "files",
					Value: fs,
				},
			},
		}
		args, _ := cli.Split("app --no-recursive .")
		_ = app.RunContext(context.Background(), args)
		Expect(fs.Recursive).To(BeFalse())
	})

	It("invokes bind func with corresponding value", func() {
		var calledWith string
		fs := new(cli.FileSet)
		act := new(joeclifakes.FakeAction)
		call := func(_ *cli.FileSet, s string) error {
			calledWith = s
			return nil
		}
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name:   "recursive",
					Value:  new(string),
					Action: act,
					Uses:   bind.Indirect("files", call),
				},
			},
			Args: []*cli.Arg{
				{
					Name:  "files",
					Value: fs,
				},
			},
		}
		args, _ := cli.Split("app --recursive YES .")
		_ = app.RunContext(context.Background(), args)
		Expect(act.ExecuteCallCount()).To(Equal(1), "action should still be called")
		Expect(calledWith).To(Equal("YES"))
	})
})

var _ = Describe("Redirect", func() {

	It("copies the implied value of the function", func() {
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name: "t",
				},
				{
					Name: "u",
					Uses: bind.Redirect[uint64]("t"),
				},
			},
		}
		app.Initialize(context.Background())
		Expect(app.Flags[0].Value).To(Equal(new(uint64)))
		Expect(app.Flags[1].Value).To(Equal(new(uint64)))
	})

	It("invokes bind func with static value", func() {
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name: "t",
				},
				{
					Name: "u",
					Uses: bind.Redirect[uint64]("t", 420),
				},
			},
		}

		args, _ := cli.Split("app -u")
		_ = app.RunContext(context.Background(), args)
		Expect(app.Flags[0].Value).To(PointTo(Equal(uint64(420))))
		Expect(app.Flags[1].Value).To(PointTo(Equal(true)))
	})
})
