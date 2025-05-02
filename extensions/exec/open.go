// Copyright 2022 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//go:build !windows && !darwin

package exec

import (
	"os"
	"os/exec"
)

func openUsingApp(input, app string) (*exec.Cmd, error) {
	if os.Getenv("DISPLAY") != "" {
		return exec.Command("xdg-open", input), nil
	}

	return nil, nil
}
