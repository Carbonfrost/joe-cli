package support

import (
	"fmt"
	"regexp"

	"github.com/Carbonfrost/joe-cli/internal/privatekey"
)

var validIdentifierPattern = regexp.MustCompile(`^[a-zA-Z0-9@#+\._\*:\?-]+$`)

// targetDataAccessor is the interface target and expr.Expr use
type targetDataAccessor interface {
	SetData(name any, v any)
	LookupData(name any) (any, bool)
}

func PromoteOptionalAliases(data targetDataAccessor, aliases *[]string, names map[string]bool) {
	const optionalAliasesDataKey = privatekey.OptionalAliases
	dd, ok := data.LookupData(optionalAliasesDataKey)
	if !ok {
		return
	}
	if optionalAliases, ok := dd.([]string); ok {
		for _, alias := range optionalAliases {
			if !names[alias] {
				*aliases = append(*aliases, alias)
				names[alias] = true
			}
		}
		data.SetData(optionalAliasesDataKey, nil)
	}
}

func PrivateData(public *map[string]any, private map[any]any) PD {
	if *public == nil {
		*public = map[string]any{}
	}
	return PD{public: *public, private: private}
}

type PD struct {
	public  map[string]any // a reference to Data
	private map[any]any
}

func (p PD) Lookup(key any) (any, bool) {
	if name, ok := key.(string); ok {
		value, ok := p.public[name]
		return value, ok
	}
	value, ok := p.private[key]
	return value, ok
}

func (p PD) Set(key, value any) {
	if name, ok := key.(string); ok {
		p.public[name] = value
	} else {
		p.private[key] = value
	}
}

// ValidateNames aggregates names into the specified map and looks for duplicates
func ValidateNames(names map[string]bool, name string, aliases []string, checkPersistent func(name string) string) (errs []error) {
	if checkPersistent == nil {
		checkPersistent = nilStrStr
	}
	if err := checkValidFlagIdentifier(name); err != nil {
		errs = append(errs, err)
	} else if names[name] {
		errs = append(errs, fmt.Errorf("duplicate name used: %q%s", name, checkPersistent(name)))
	}
	for _, a := range aliases {
		if err := checkValidFlagIdentifier(a); err != nil {
			errs = append(errs, fmt.Errorf("invalid alias %q%s: %w", a, checkPersistent(name), err))
		} else if names[a] {
			errs = append(errs, fmt.Errorf("duplicate name used: %q%s", a, checkPersistent(name)))
		}
		names[a] = true
	}
	names[name] = true
	return
}

func checkValidFlagIdentifier(name string) error {
	if !validIdentifierPattern.MatchString(name) {
		return fmt.Errorf("not a valid name")
	}
	return nil
}

func nilStrStr(string) string {
	return ""
}
