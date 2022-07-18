package template

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/Carbonfrost/joe-cli"
)

type Context struct {
	*cli.Context

	Vars      map[string]interface{}
	Overwrite bool
	DryRun    bool

	working []string
}

var (
	colors = map[string]cli.Color{
		"error":     cli.Red,
		"create":    cli.Green,
		"overwrite": cli.Cyan,
		"identical": cli.Gray,
	}
	padding = strings.Repeat(" ", 12)
)

func (c *Context) Do(gens ...Generator) error {
	for _, g := range gens {
		err := g.Generate(c)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Context) Exists(name string) bool {
	_, err := c.Stat(name)
	return err == nil || !errors.Is(err, fs.ErrNotExist)
}

func (c *Context) File(name string) string {
	if strings.HasPrefix(name, "/") {
		return name
	}
	return filepath.Join(c.WorkDir(), c.expandName(name))
}

// WorkDir is the path to the working directory
func (c *Context) WorkDir() string {
	return filepath.Clean(filepath.Join(c.working...))
}

func (c *Context) PushDir(name string) error {
	c.working = append(c.working, name)
	return nil
}

func (c *Context) PopDir() error {
	c.working = c.working[0 : len(c.working)-1]
	if len(c.working) == 0 {
		return fmt.Errorf("cannot pop dir")
	}
	return nil
}

func (c *Context) SetData(name string, value interface{}) {
	c.Vars[name] = value
}

func (c *Context) error(file string) {
	c.trace("error", file)
}

func (c *Context) create(file string) {
	c.trace("create", file)
}

func (c *Context) identical(file string) {
	c.trace("identical", file)
}

func (c *Context) overwrite(file string) {
	c.trace("overwrite", file)
}

func (c *Context) trace(category string, file string) {
	color, ok := colors[category]
	if !ok {
		color = cli.Default
	}
	out := c.Context.Stdout
	fmt.Fprint(out, padding[0:len(padding)-len(category)])
	out.SetForeground(color)
	fmt.Fprint(out, category)
	out.Reset()
	fmt.Fprint(out, "  ")

	fmt.Fprint(out, file)
	if c.DryRun {
		fmt.Fprint(out, " (dry-run)")
	}
	fmt.Fprintln(out)
}

func (c *Context) reportChange(original []byte, name string, created bool) {
	if created {
		c.create(name)
		return
	}

	newContents, _ := func() ([]byte, error) {
		f, err := c.actualFS().Open(c.File(name))
		if err != nil {
			return nil, nil
		}
		return io.ReadAll(f)
	}()
	if bytes.Equal(original, newContents) {
		c.identical(name)
	} else {
		c.overwrite(name)
	}
}

func (c *Context) expandName(name string) string {
	var buf bytes.Buffer
	tpl, err := template.New("fileName").Parse(name)
	if err != nil {
		return name
	}
	err = tpl.Execute(&buf, c.Vars)
	if err != nil {
		return name
	}
	return buf.String()
}

func (c *Context) Stat(name string) (fs.FileInfo, error) {
	return c.actualFS().Stat(c.File(name))
}

func (c *Context) Open(name string) (fs.File, error) {
	return c.actualFS().Open(c.File(name))
}

func (c *Context) Chmod(name string, mode fs.FileMode) error {
	name, err := c.pathEnsure(name)
	if err != nil {
		return err
	}
	return c.actualFS().Chmod(name, mode)
}

func (c *Context) Chown(name string, uid, gid int) error {
	name, err := c.pathEnsure(name)
	if err != nil {
		return err
	}
	return c.actualFS().Chown(name, uid, gid)
}

func (c *Context) Create(name string) (fs.File, error) {
	name, err := c.pathEnsure(name)
	if err != nil {
		return nil, err
	}
	return c.actualFS().Create(name)
}

func (c *Context) Mkdir(name string, perm fs.FileMode) error {
	name, err := c.pathEnsure(name)
	if err != nil {
		return err
	}
	return c.actualFS().Mkdir(name, perm)
}

func (c *Context) MkdirAll(path string, perm fs.FileMode) error {
	return c.actualFS().MkdirAll(c.File(path), perm)
}

func (c *Context) OpenFile(name string, flag int, perm fs.FileMode) (fs.File, error) {
	name = c.File(name)
	return c.actualFS().OpenFile(name, flag, perm)
}

func (c *Context) Remove(name string) error {
	name = c.File(name)
	return c.actualFS().Remove(name)
}

func (c *Context) RemoveAll(path string) error {
	return c.actualFS().RemoveAll(c.File(path))
}

func (c *Context) Chtimes(name string, atime time.Time, mtime time.Time) error {
	name, err := c.pathEnsure(name)
	if err != nil {
		return err
	}
	return c.actualFS().Chtimes(name, atime, mtime)
}

func (c *Context) Rename(oldpath, newpath string) error {
	oldpath, err := c.pathEnsure(oldpath)
	if err != nil {
		return err
	}

	newpath, err = c.pathEnsure(newpath)
	if err != nil {
		return err
	}

	return c.actualFS().Rename(oldpath, newpath)
}

func (c *Context) pathEnsure(name string) (string, error) {
	name = c.expandName(name)
	if strings.HasPrefix(name, "/") {
		return name, nil
	}

	dir := filepath.Dir(name)
	if _, err := c.actualFS().Stat(dir); os.IsNotExist(err) {
		err = c.actualFS().MkdirAll(dir, 0755)
		if err != nil {
			return c.File(name), err
		}
	}

	return c.File(name), nil
}

func (c *Context) actualFS() cli.FS {
	return c.FS.(cli.FS)
}

var (
	_ cli.FS = (*Context)(nil)
)
