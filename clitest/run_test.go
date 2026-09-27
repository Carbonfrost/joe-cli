// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package clitest_test

import (
	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/clitest"
	joeclifakes "github.com/Carbonfrost/joe-cli/internal/joe-clifakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cmd", func() {
	Describe("Run", func() {

		It("runs the underlying app", func() {
			action := new(joeclifakes.FakeAction)
			app := cli.NewApp(&cli.Command{
				Action: action,
			})

			Expect(clitest.Command(app, "app").Run()).To(Succeed())
			Expect(action.ExecuteCallCount()).To(Equal(1))
		})
	})

	Describe("Args", func() {

		It("gets the args that were set", func() {
			app := cli.NewApp(&cli.Command{})
			cmd := clitest.Command(app, "app", "1", "2")

			Expect(cmd.Args()).To(Equal([]string{"app", "1", "2"}))
			Expect(cmd.String()).To(Equal("app 1 2"))
		})
	})

})
