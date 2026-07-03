package config

import (
	"iter"
	"maps"
)

// EnvProvider provides the environment of a workspace, which
// is generally a set of environment variables that the workspace
// either depends upon or implicitly defines for its configuration
// behavior.
type EnvProvider interface {

	// Environ obtains the environment variables
	Environ() iter.Seq2[string, string]
}

// EnvMap provides a basic implementation of EnvProvider using
// a map
type EnvMap map[string]string

// Environ obtains the environment variables from the map
func (m EnvMap) Environ() iter.Seq2[string, string] {
	return maps.All(m)
}

// WithEnvProvider sets the environment provider on the workspace
func WithEnvProvider(env EnvProvider) WorkspaceOption {
	return workspaceOption(func(w *Workspace) error {
		w.env = env
		return nil
	})
}
