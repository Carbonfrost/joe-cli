// Copyright 2022 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//go:build darwin

package exec

import (
	"os/exec"
)

func openUsingApp(input, app string) (*exec.Cmd, error) {
	args := []string{input}
	if app != "" {
		args = append(args, "-a", app)
	}
	cmd := exec.Command("/usr/bin/open", args...)
	return cmd, nil
}
