//go:build tools

package tools

import (
	_ "github.com/maxbrunsfeld/counterfeiter/v6"
	_ "github.com/onsi/ginkgo/v2"
    _ "github.com/nishanths/exhaustive"
    _ "honnef.co/go/tools/cmd/staticcheck"
)
