package cli

import (
	"fmt"
	"strings"
)

type Value interface {
	Set(string) error
	String() string
}

type boolValue bool
type stringValue string

func (b *boolValue) Set(value string) error {
	switch strings.ToLower(value) {
	case "", "1", "true", "on", "t":
		*b = true
	case "0", "false", "off", "f":
		*b = false
	default:
		return fmt.Errorf("invalid value for bool %q", value)
	}
	return nil
}

func (b *boolValue) String() string {
	if *b {
		return "true"
	}
	return "false"
}

func (s *stringValue) Set(value string) error {
	*s = stringValue(value)
	return nil
}

func (s *stringValue) String() string {
	return string(*s)
}
