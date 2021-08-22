package cli

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/kballard/go-shellquote"
)

var thisPackage string

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

func Split(s string) ([]string, error) {
	return shellquote.Split(s)
}

func init() {
	pc, _, _, ok := runtime.Caller(0)
	if !ok {
		return
	}
	f := runtime.FuncForPC(pc)
	if f == nil {
		return
	}
	thisPackage = f.Name()
	x := strings.LastIndex(thisPackage, "/")
	if x < 0 {
		return
	}
	y := strings.Index(thisPackage[x:], ".")
	if y < 0 {
		return
	}
	// thisPackage includes the trailing . after the package name.
	thisPackage = thisPackage[:x+y+1]
}

func calledFrom() string {
	for i := 2; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			return ""
		}
		if !strings.HasSuffix(file, "_test.go") {
			f := runtime.FuncForPC(pc)
			if f != nil && strings.HasPrefix(f.Name(), thisPackage) {
				continue
			}
		}
		return fmt.Sprintf("%s:%d", file, line)
	}
}
