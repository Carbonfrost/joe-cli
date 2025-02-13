package cli

import (
	"context"
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

	subcommandDidNotExecute

	// Anything matches any kind of target
	Anything = AnyFlag | AnyArg | AnyCommand
)

var (
	filterModeMatches = map[FilterModes]func(*Context) bool{
		Anything:                anyImpl,
		AnyFlag:                 (*Context).IsFlag,
		AnyArg:                  (*Context).IsArg,
		AnyCommand:              (*Context).IsCommand,
		HasValue:                (*Context).HasValue,
		Seen:                    seenThis,
		RootCommand:             isRoot,
		subcommandDidNotExecute: (*Context).subcommandDidNotExecute,
	}

	filterModeLabels = map[FilterModes][2]string{
		AnyFlag:     {"any flag", "ANY_FLAG"},
		AnyArg:      {"any arg", "ANY_ARG"},
		AnyCommand:  {"any command", "ANY_COMMAND"},
		Anything:    {"anything", "ANYTHING"},
		Seen:        {"option that has been seen", "SEEN"},
		RootCommand: {"root command", "ROOT_COMMAND"},
		HasValue:    {"target with value", "HAS_VALUE"},
	}
)

func anyImpl(*Context) bool    { return true }
func seenThis(c *Context) bool { return c.Seen("") }
func isRoot(c *Context) bool   { return c.Parent() == nil }

// PatternFilter parses a context pattern string and returns
// a filter which matches on it.
func PatternFilter(pat string) ContextFilter {
	return newContextPathPattern(pat)
}

func (f FilterModes) Matches(ctx context.Context) bool {
	// Note that Anything has the highest hamming weight so it gets
	// tested first using the optimal implementation
	c := FromContext(ctx)
	for _, match := range decompose(filterModeMatches).items(f) {
		if !match(c) {
			return false
		}
	}
	return true
}

// String produces a textual representation of the timing
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

// Describe produces a representation of the timing suitable for use in messages
func (f FilterModes) Describe() string {
	return filterModeLabels[f][0]
}

func (f ContextFilterFunc) Matches(c context.Context) bool {
	if f == nil {
		return true
	}
	return f(FromContext(c))
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
