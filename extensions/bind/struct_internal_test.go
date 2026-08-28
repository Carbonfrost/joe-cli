// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bind

import (
	// The ginkgo DSL is not dot-imported because Context is declared by this package
	ginkgo "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("inflectName", func() {

	ginkgo.DescribeTable("examples", func(name string, expected string) {
		Expect(inflectName(name)).To(Equal(expected))
	},
		ginkgo.Entry("empty", "", ""),
		ginkgo.Entry("single letter", "V", "v"),
		ginkgo.Entry("single word", "Verbose", "verbose"),
		ginkgo.Entry("two words", "ConfigFile", "config-file"),
		ginkgo.Entry("three words", "MaximumFileSize", "maximum-file-size"),
		ginkgo.Entry("acronym", "URL", "url"),
		ginkgo.Entry("acronym plural", "IDs", "ids"),
		ginkgo.Entry("acronym then word", "HTTPPort", "http-port"),
		ginkgo.Entry("word then acronym", "FileID", "file-id"),
		ginkgo.Entry("acronym within name", "AWSAccessKeyID", "aws-access-key-id"),
		ginkgo.Entry("acronym with digit", "IPv4Address", "ipv4-address"),
		ginkgo.Entry("digit within word", "Level2Cache", "level2-cache"),
		ginkgo.Entry("single letter word", "DoAThing", "do-a-thing"),
		ginkgo.Entry("underscore delimiter", "Config_File", "config-file"),
		ginkgo.Entry("leading underscore", "_Config", "config"),
		ginkgo.Entry("redundant delimiters", "Config__File", "config-file"),
		ginkgo.Entry("already inflected", "config-file", "config-file"),
	)
})
