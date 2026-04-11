package support

import (
	"fmt"
	"regexp"
)

var validIdentifierPattern = regexp.MustCompile(`^[a-zA-Z0-9@#+\._\*:\?-]+$`)

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
