// Copyright 2022, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cli_test

import (
	"errors"
	"flag"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGocli(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gocli Suite")
}

type temperature string

func (t *temperature) Set(s string) error {
	switch s {
	case "F":
		*t = "Fahrenheit"
	case "C":
		*t = "Celsius"
	default:
		return errors.New("not supported")
	}
	return nil
}

func (t *temperature) String() string {
	return string(*t)
}

func (*temperature) Synopsis() string {
	return "{Fahrenheit|Celsius}"
}

type boolFlag string

func (*boolFlag) Set(s string) error {
	return nil
}

func (*boolFlag) IsBoolFlag() bool {
	return true
}

func (*boolFlag) String() string {
	return ""
}

var _ flag.Value = (*temperature)(nil)
var _ flag.Value = (*boolFlag)(nil)
