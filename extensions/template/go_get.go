package template

import (
	"os"
	"os/exec"
	"strings"
)

// GoGet assumes that the current template is a Go module and adds the given go module via
// go get.  An error results if the project is not a go module.
func GoGet(pkgs ...string) Generator {
	return &goGetter{pkgs}
}

type goGetter struct {
	pkgs []string
}

func (d *goGetter) Generate(c *Context) error {
	if c.DryRun {
		return d.dryRun(c)
	}
	return d.realGenerate(c)
}

func (d *goGetter) realGenerate(c *Context) error {
	originalMod, originalSum := d.files()

	err := execGoGet(d.pkgs)
	if err != nil {
		c.error("go.mod")
		return err
	}

	c.reportChange(originalMod, "go.mod", false)
	c.reportChange(originalSum, "go.sum", false)
	return nil
}

func (d *goGetter) dryRun(c *Context) error {
	originalMod, _ := d.files()

	for _, pkg := range d.pkgs {
		if !strings.Contains(string(originalMod), pkg) {
			c.overwrite("go.mod")
			c.overwrite("go.sum")
			return nil
		}
	}
	return nil
}

func (d *goGetter) files() (originalMod, originalSum []byte) {
	originalMod, _ = os.ReadFile("go.mod")
	originalSum, _ = os.ReadFile("go.sum")
	return
}

func execGoGet(modules []string) error {
	args := append([]string{"get"}, modules...)
	cmd := exec.Command("go", args...)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
