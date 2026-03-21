// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cli

import (
	"context"
	"fmt"
	"math/bits"
	"sort"
	"strings"
)

// ContextFilterFunc provides a predicate function which detects whether the context
// applies.
type ContextFilterFunc func(*Context) bool

// ContextFilter is used to implement logic for filtering on matching
// contexts.  The main use case is conditional actions using the IfMatch
// decorator.
type ContextFilter interface {
	Matches(c context.Context) bool
}

// FilterModes enumerates common context filters.  These are bitwise-combinable.
type FilterModes int

type bitSet[T ~int, V any] struct {
	keys []uint
	m    map[T]V
}

type (
	anyFilter []ContextFilter
	allFilter []ContextFilter
)

type hasDataFilter struct {
	name     string
	value    any
	hasValue bool
}

const (
	// AnyFlag filters the context for any flag
	AnyFlag FilterModes = 1 << iota

	// AnyArg filters the context for any arg
	AnyArg

	// AnyCommand filters the context for any command
	AnyCommand

	// Seen filters the context to detect if the current flag or arg has been seen
	Seen

	// RootCommand filters the context to detect if the current command is the root command
	RootCommand

	// HasValue checks whether the target is an arg, flag, or value setup
	HasValue

	// Completing checks whether completion is occurring
	Completing

	// Defines filters the context to detect if the current flag or arg was defined by Joe-cli package
	Defines

	subcommandDidNotExecute

	// Anything matches any kind of target
	Anything = AnyFlag | AnyArg | AnyCommand
)

var (
	filterModeMatches = map[FilterModes]ContextFilter{
		Anything:                ContextFilterFunc(anyImpl),
		AnyFlag:                 ContextFilterFunc((*Context).IsFlag),
		AnyArg:                  ContextFilterFunc((*Context).IsArg),
		AnyCommand:              ContextFilterFunc((*Context).IsCommand),
		HasValue:                ContextFilterFunc((*Context).HasValue),
		Seen:                    ContextFilterFunc(seenThis),
		RootCommand:             ContextFilterFunc(isRoot),
		Completing:              ContextFilterFunc(completingThis),
		Defines:                 HasData(SourceAnnotation()),
		subcommandDidNotExecute: ContextFilterFunc((*Context).subcommandDidNotExecute),
	}

	filterModeLabels = map[FilterModes][2]string{
		AnyFlag:     {"any flag", "ANY_FLAG"},
		AnyArg:      {"any arg", "ANY_ARG"},
		AnyCommand:  {"any command", "ANY_COMMAND"},
		Anything:    {"anything", "ANYTHING"},
		Seen:        {"option that has been seen", "SEEN"},
		RootCommand: {"root command", "ROOT_COMMAND"},
		HasValue:    {"target with value", "HAS_VALUE"},
		Completing:  {"in shell completion", "COMPLETING"},
		Defines:     {"defined in joe-cli pkg", "DEFINES"},
	}
)

func anyImpl(*Context) bool          { return true }
func seenThis(c *Context) bool       { return c.Seen("") }
func isRoot(c *Context) bool         { return c.Parent() == nil }
func completingThis(c *Context) bool { return c.robustParsingMode() }

// Any provides a composite ContextFilter where any filter
// from a list can match. When empty, this is the same as Anything.
func Any(f ...ContextFilter) ContextFilter {
	return castUniverseFilter[anyFilter](f)
}

// All provides a composite ContextFilter where all filters
// from a list must match. When empty, this is the same as Anything.
func All(f ...ContextFilter) ContextFilter {
	return castUniverseFilter[allFilter](f)
}

// HasFlag provides a context filter that detects whether a flag exists
func HasFlag(name any) ContextFilter {
	return findByNameFilter(name, (*Context).LocalFlags, findFlagByName)
}

// HasArg provides a context filter that detects whether an arg exists
func HasArg(name any) ContextFilter {
	return findByNameFilter(name, (*Context).LocalArgs, findArgByName)
}

// HasCommand provides a context filter that detects whether a command exists
func HasCommand(name any) ContextFilter {
	return findByNameFilter(name, (*Context).LocalCommands, findCommandByName)
}

func findByNameFilter[T any](name any, fn func(*Context) []T, finder func([]T, any) (T, int, bool)) ContextFilterFunc {
	return func(c *Context) bool {
		_, _, ok := finder(fn(c), name)
		return ok
	}
}

// PatternFilter parses a context pattern string and returns
// a filter which matches on it.
func PatternFilter(pat string) ContextFilter {
	var result []ContextFilter

	for expr := range strings.SplitSeq(pat, ",") {
		result = append(result, newContextPathExpr(strings.TrimSpace(expr)))
	}
	return Any(result...)
}

func newContextPathExpr(expr string) ContextFilter {
	if kvp, ok := strings.CutPrefix(expr, "{"); ok {
		if kvp, ok = strings.CutSuffix(kvp, "}"); ok {
			key, value, hasValue := strings.Cut(kvp, ":")
			if hasValue {
				return HasData(strings.TrimSpace(key), strings.TrimSpace(value))
			}

			return HasData(strings.TrimSpace(key))
		}
	}
	return contextPathPattern{strings.Fields(expr)}
}

// HasSeen provides a context filter for when a particular flag or arg has been seen
func HasSeen(name any) ContextFilter {
	if name == "" || name == nil {
		return Seen
	}
	return ContextFilterFunc(func(c *Context) bool {
		return c.Seen(name)
	})
}

// HasData provides a context filter that detects whether certain
// data is in the context.  Optionally, the value can be checked.
func HasData(name string, valueopt ...any) ContextFilter {
	switch len(valueopt) {
	case 1:
		return hasDataFilter{name, valueopt[0], true}

	case 0:
		return hasDataFilter{name, nil, false}
	default:
		panic("valueopt must specify either zero or one values")
	}
}

func (h hasDataFilter) Matches(c context.Context) bool {
	fn := func(any) bool { return true }
	if h.hasValue {
		fn = func(val any) bool {
			return val == h.value
		}
	}

	value, ok := FromContext(c).LookupData(h.name)
	return ok && fn(value)
}

func (h hasDataFilter) String() string {
	if !h.hasValue {
		return fmt.Sprintf("{%s}", h.name)
	}
	if value, ok := h.value.(string); ok {
		return fmt.Sprintf("{%s:%s}", h.name, value)
	}
	return fmt.Sprintf("{%v %v}", h.name, h.value)
}

// Matches detects whether the context matches the filter modes
func (f FilterModes) Matches(ctx context.Context) bool {
	// Note that Anything has the highest hamming weight so it gets
	// tested first using the optimal implementation
	c := FromContext(ctx)
	for _, match := range decompose(filterModeMatches).items(f) {
		if !match.Matches(c) {
			return false
		}
	}
	return true
}

// String produces a textual representation of the filter modes
func (f FilterModes) String() string {
	return filterModeLabels[f][1]
}

// MarshalText provides the textual representation
func (f FilterModes) MarshalText() ([]byte, error) {
	return []byte(f.String()), nil
}

// UnmarshalText converts the textual representation
func (f *FilterModes) UnmarshalText(b []byte) error {
	token := strings.TrimSpace(string(b))
	for k, y := range filterModeLabels {
		if token == y[1] {
			*f = k
			return nil
		}
	}
	return nil
}

// Describe produces a representation of the filter modes suitable for use in messages
func (f FilterModes) Describe() string {
	return filterModeLabels[f][0]
}

// Matches implements [ContextFilter]
func (f ContextFilterFunc) Matches(c context.Context) bool {
	if f == nil {
		return true
	}
	return f(FromContext(c))
}

func (a anyFilter) Matches(c context.Context) bool {
	for _, f := range a {
		if f.Matches(c) {
			return true
		}
	}
	return false
}

func (a allFilter) Matches(c context.Context) bool {
	for _, f := range a {
		if !f.Matches(c) {
			return false
		}
	}
	return true
}

func castUniverseFilter[TFilter interface {
	~[]ContextFilter
	ContextFilter
}](f []ContextFilter) ContextFilter {
	if len(f) == 0 {
		return Anything
	}
	if len(f) == 1 {
		return f[0]
	}
	return TFilter(f)
}

func decompose[T ~int, V any](m map[T]V) *bitSet[T, V] {
	var i int

	keys := make([]uint, len(m))
	for k := range m {
		keys[i] = uint(k)
		i++
	}
	sort.Slice(keys, func(i, j int) bool {
		if bits.OnesCount(keys[i]) > bits.OnesCount(keys[j]) {
			return true
		}
		return keys[i] > keys[j]
	})
	return &bitSet[T, V]{keys: keys, m: m}
}

func (b *bitSet[T, V]) items(values T) []V {
	options := uint(values)
	var parts []V
	for _, current := range b.keys {
		if options&current == current {
			action := b.m[T(current)]
			parts = append(parts, action)
			options &^= current
		}
	}

	return parts
}
