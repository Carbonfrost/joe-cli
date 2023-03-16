package support

import (
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/term"
)

func ParseMap(values []string) map[string]string {
	res := map[string]string{}

	var key, value string
	for _, kvp := range values {
		k := SplitList(kvp, "=", 2)
		switch len(k) {
		case 2:
			key = k[0]
			value = k[1]
		case 1:
			// Implies comma was meant to be escaped
			// -m key=value,s,t  --> interpreted as key=value,s,t rather than s and t keys
			value = value + "," + k[0]
		}
		res[key] = value
	}
	return res
}

func FormatMap(m map[string]string, delim string) string {
	items := make([]string, len(m))
	var i int
	for k, v := range m {
		items[i] = k + "=" + v
		i++
	}
	sort.Strings(items)
	return strings.Join(items, delim)
}

func SplitList(s, sep string, n int) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if strings.Contains(s, "\\") {
		regex := regexp.MustCompile(`(^|[^\\])` + regexp.QuoteMeta(sep))
		matches := regex.FindAllStringSubmatchIndex(s, n)

		if len(matches) == 0 {
			return []string{s}
		}

		unquote := func(x string) string {
			return strings.ReplaceAll(x, "\\", "")
		}
		res := make([]string, 0)

		var last int
		for _, match := range matches {
			res = append(res, unquote(s[last:match[1]-1]))
			res = append(res, unquote(s[match[2]+1+1:]))
			last = match[2] + 1 + 1
		}
		return res
	}
	return strings.SplitN(s, sep, n)
}

func GuessWidth() int {
	cols := os.Getenv("COLUMNS")
	if cols != "" {
		width, err := strconv.Atoi(cols)
		if err == nil && width > 12 && width < 80 {
			return width
		}
	}

	fd := int(os.Stdout.Fd())
	if term.IsTerminal(fd) {
		width, _, err := term.GetSize(fd)
		if err == nil && width > 12 && width < 80 {
			return width
		}
	}
	return 80
}
