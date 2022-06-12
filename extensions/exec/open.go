//go:build !windows && !darwin

package exec

import (
	"os"
	"os/exec"
)

func openUsingApp(input, app string) *exec.Cmd {
	if os.Getenv("DISPLAY") != "" {
		return exec.Command("xdg-open", input)
	}

	return nil
}
