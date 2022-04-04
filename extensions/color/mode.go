package color

import (
	"errors"
	"flag"
	"strings"
)

// Mode enumerates the color mode: on, off, or auto-detect
type Mode int

const (
	// Auto will enable color depending upon whether stdout is detected as TTY.
	Auto = iota

	// Always enable terminal color
	Always

	// Never enable terminal color
	Never
)

func (*Mode) Synopsis() string {
	return "{auto|always|never}"
}

func (m *Mode) Set(arg string) error {
	switch strings.ToLower(arg) {
	case "auto":
		*m = Auto
	case "always", "true", "on", "":
		*m = Always
	case "never", "false", "off":
		*m = Never
	default:
		return errors.New("invalid value")
	}
	return nil
}

func (m Mode) String() string {
	switch m {
	case Never:
		return "never"
	case Always:
		return "always"
	case Auto:
	default:
	}
	return "auto"
}

var _ flag.Value = (*Mode)(nil)
