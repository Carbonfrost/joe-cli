// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package reservedoptions identifies the options which are shared among extensions.
package reservedoptions

import (
	cli "github.com/Carbonfrost/joe-cli"
)

// The reserved options are shared between extensions and must be globally unique.

const (
	ExprParseAllowInlineValues = cli.ReservedOption1
	ExprParseOperators         = cli.ReservedOption2
)
