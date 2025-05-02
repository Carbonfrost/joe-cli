// Copyright 2022 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//go:build windows

package exec

import (
	"os"
	eexec "os/exec"
	"path/filepath"
)

var (
	cmd      = "url.dll,FileProtocolHandler"
	runDll32 = filepath.Join(os.Getenv("SYSTEMROOT"), "System32", "rundll32.exe")
)

func openUsingApp(input, app string) (*eexec.Cmd, error) {
	cmd := eexec.Command(runDll32, cmd, input)
	return cmd, nil
}
