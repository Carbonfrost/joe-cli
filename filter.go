package cli

import (
	"math/bits"
	"sort"
)

// ContextFilterFunc provides a predicate function which detects whether the context
// applies.
type ContextFilterFunc func(*Context) bool

// ContextFilter is used to implement logic for filtering on matching
// contexts.  The main use case is conditional actions using the IfMatch
// decorator.
type ContextFilter interface {
	Matches(c *Context) bool
}

// FilterModes enumerates common context filters.  These are bitwise-combinable
// including with Timing.
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

	// Anything matches any kind of target
	Anything = AnyFlag | AnyArg | AnyCommand
)

var (
	filterModeMatches = map[FilterModes]func(*Context) bool{
		Anything:    anyImpl,
		AnyFlag:     (*Context).IsFlag,
		AnyArg:      (*Context).IsArg,
		AnyCommand:  (*Context).IsCommand,
		Seen:        seenThis,
		RootCommand: isRoot,
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

func (f FilterModes) Matches(c *Context) bool {
	// Note that Anything has the highest hamming weight so it gets
	// tested first using the optimal implementation
	for _, match := range decompose(filterModeMatches).items(f) {
		if !match(c) {
			return false
		}
	}
	return true
}

func (f ContextFilterFunc) Matches(c *Context) bool {
	if f == nil {
		return true
	}
	return f(c)
}

func decompose[T ~int, V any](m map[T]V) *bitSet[T, V] {
	var i int

	keys := make([]uint, len(m))
	for k := range m {
		keys[i] = uint(k)
		i++
	}
	sort.Slice(keys, func(i, j int) bool {
		return bits.OnesCount(keys[i]) > bits.OnesCount(keys[j])
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
