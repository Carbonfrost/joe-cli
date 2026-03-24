// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec_test

import (
	"context"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/exec"
	joeclifakes "github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HaveLookPath", func() {

	It("finds the path", func() {
		fakeAction := new(joeclifakes.FakeAction)
		app := &cli.App{
			Name:   "app",
			Action: cli.IfMatch(exec.HaveLookPath("go"), fakeAction),
		}
		_ = app.RunContext(context.Background(), nil)

		Expect(fakeAction.ExecuteCallCount()).To(Equal(1))
	})

})
