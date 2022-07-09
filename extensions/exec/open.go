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
