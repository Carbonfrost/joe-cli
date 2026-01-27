// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Suite")
}
