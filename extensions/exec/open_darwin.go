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
