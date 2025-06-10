// Copyright 2022, 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package internal

import (
	"runtime/debug"
)

var Version string

func init() {
	info, _ := debug.ReadBuildInfo()
	Version = info.Main.Version
}
