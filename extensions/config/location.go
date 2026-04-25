// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Location defines a representation of configuration file locations
type Location interface {
	// Paths resolves the file paths for this location
	Paths(context.Context) ([]string, error)
}

// IdiomaticLocation defines a location which exists in a particular layer
type IdiomaticLocation interface {
	Location

	// Layer gets the layer where the location is found
	Layer() Layer

	// OS gets the operating system the location supports
	OS() string

	// Arch gets the architecture the location supports
	Arch() string
}

// IdiomaticLocationProvider provides an interface for how idiomatic locations
// are detected
type IdiomaticLocationProvider interface {
	Location

	// Resolve determines the locations
	Resolve(Options) ([]IdiomaticLocation, error)

	// FindProfileNames locates available profile names
	FindProfileNames(context.Context) []string
}

// Layer identifies layers in the configuration system
type Layer int

// Layers where a location exists
const (
	LayerUnspecified Layer = -1
	LayerIntrinsic   Layer = 0
	LayerSystem      Layer = 2
	LayerUser        Layer = 4
	LayerWorkspace   Layer = 6
	LayerProfile     Layer = 8
	LayerAdditional  Layer = 10
)

var (
	layerLabels = map[Layer]string{
		LayerUnspecified: "UNSPECIFIED",
		LayerIntrinsic:   "INTRINSIC",
		LayerSystem:      "SYSTEM",
		LayerUser:        "USER",
		LayerWorkspace:   "WORKSPACE",
		LayerProfile:     "PROFILE",
		LayerAdditional:  "ADDITIONAL",
	}
)

func (l Layer) String() string {
	l = max(LayerUnspecified, min(LayerAdditional, l))
	if res, ok := layerLabels[l]; ok {
		return res
	}
	if res, ok := layerLabels[l-1]; ok {
		return res + "+1"
	}
	return strconv.Itoa(int(l))
}

// ParseLocation parses a location string and returns the corresponding Location implementation.
// The location type is determined by the suffix of the input string:
//
//   - Ending with "/" indicates a directory location (enumerates files in the directory)
//   - Ending with "/..." indicates a directory tree location (walks entire hierarchy)
//   - No special suffix indicates a file location
//
// All locations support environment variable expansion. In addition to OS env vars,
// these special variables are supported:
//
//   - cli:app - the app name
//   - cli:workspace - the workspace directory
//   - cli:workspace.config - the workspace ConfigDir directory
//   - GOOS - populated from runtime.GOOS
//   - GOARCH - populated from runtime.GOARCH
//
// If a variable cannot be resolved (e.g., no workspace in context), the path will not
// be yielded from Paths.
func ParseLocation(s string) Location {
	if strings.HasSuffix(s, "/...") {
		return &directoryTreeLocation{path: strings.TrimSuffix(s, "...")}
	}
	if strings.HasSuffix(s, "/") {
		return &directoryLocation{path: s}
	}
	return &fileLocation{path: s}
}

type fileLocation struct {
	path string
}

func (f *fileLocation) Paths(ctx context.Context) ([]string, error) {
	expanded, ok := expandPath(ctx, f.path)
	if !ok {
		return nil, nil
	}
	return []string{expanded}, nil
}

type directoryLocation struct {
	path string
}

func (d *directoryLocation) Paths(ctx context.Context) ([]string, error) {
	expanded, ok := expandPath(ctx, d.path)
	if !ok {
		return nil, nil
	}

	entries, err := os.ReadDir(expanded)
	if err != nil {
		return nil, err
	}

	var paths []string
	for _, entry := range entries {
		if !entry.IsDir() {
			paths = append(paths, filepath.Join(expanded, entry.Name()))
		}
	}
	return paths, nil
}

type directoryTreeLocation struct {
	path string
}

func (dt *directoryTreeLocation) Paths(ctx context.Context) ([]string, error) {
	expanded, ok := expandPath(ctx, dt.path)
	if !ok {
		return nil, nil
	}

	var paths []string
	err := filepath.WalkDir(expanded, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return paths, nil
}

func expandPath(ctx context.Context, path string) (string, bool) {
	allResolved := true
	expanded := os.Expand(path, func(varName string) string {
		switch varName {
		case "GOOS":
			return runtime.GOOS
		case "GOARCH":
			return runtime.GOARCH
		case "cli:app":
			name := appName(ctx)
			allResolved = allResolved && (name != "")
			return name
		case "cli:workspace":
			ws, _ := tryWorkspaceFromContext(ctx)
			allResolved = allResolved && (ws != nil && ws.Dir() != "")
			if ws == nil {
				return ""
			}
			return ws.Dir()

		case "cli:workspace.config":
			ws, _ := tryWorkspaceFromContext(ctx)
			allResolved = allResolved && (ws != nil && ws.ConfigDir() != "")
			if ws == nil {
				return ""
			}
			return ws.ConfigDir()
		default:
			if val, ok := os.LookupEnv(varName); ok {
				return val
			}
			allResolved = false
			return ""
		}
	})

	return expanded, allResolved
}
