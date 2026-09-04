// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package value_test

import (
	"time"

	"github.com/Carbonfrost/joe-cli/value"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Time", func() {

	var _ = Describe("Set", func() {

		DescribeTable("examples", func(text string, expected time.Time) {
			t := new(value.Time)
			err := t.Set(text)
			Expect(err).NotTo(HaveOccurred())
			Expect(t.Value()).To(Equal(expected))
		},
			Entry("nominal", "2026-09-03 14:25:42", time.Date(2026, 9, 3, 14, 25, 42, 0, time.Local)),
			Entry("no seconds", "2026-09-03 14:25", time.Date(2026, 9, 3, 14, 25, 0, 0, time.Local)),
			Entry("date only", "2026-09-03", time.Date(2026, 9, 3, 0, 0, 0, 0, time.Local)),
		)

	})

	Describe("Range", func() {

		It("converts to start and end of day", func() {
			t := new(value.Time)
			_ = t.Set("2010-03-11")
			start, end := t.Range()
			Expect(start).To(Equal(time.Date(2010, 3, 11, 0, 0, 0, 0, time.Local)))
			Expect(end).To(Equal(time.Date(2010, 3, 11, 23, 59, 59, 999999999, time.Local)))
		})
	})
})
