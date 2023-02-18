// Package exec allows invoking other commands, including triggering the manual, opening a document,
// or opening a Web page in the default Web browser.  It also provides a representation of the
// flag value syntax used by the -exec expression in Unix-like find designed to pass the name of
// a command and its arguments.
package exec

import (
	eexec "os/exec"
	"time"
)

// Open a file or URL, optionally in the given app.
func Open(fileapp ...string) error {
	cmd, err := func() (*eexec.Cmd, error) {
		switch len(fileapp) {
		case 1:
			return openUsingApp(fileapp[0], "")
		case 2:
			return openUsingApp(fileapp[0], fileapp[1])
		default:
			panic("invalid arguments: expected file and optional app")
		}
	}()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	return appearsSuccessful(cmd, 3*time.Second)
}

// appearsSuccessful reports whether the command appears to have run successfully.
// If the command runs longer than the timeout, it's deemed successful.
// If the command runs within the timeout, it's deemed successful if it exited cleanly.
func appearsSuccessful(cmd *eexec.Cmd, timeout time.Duration) error {
	errc := make(chan error, 1)
	go func() {
		errc <- cmd.Wait()
	}()

	select {
	case <-time.After(timeout):
		return nil
	case err := <-errc:
		return err
	}
}
