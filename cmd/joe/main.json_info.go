// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build json_info

package main

import (
	"github.com/Carbonfrost/joe-cli/internal/build"
	"github.com/Carbonfrost/joe-cli/internal/joe"
)

func main() {
	build.Dump(joe.NewApp())
}
