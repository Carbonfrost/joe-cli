package template

import (
	"bytes"
	"context"
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

type OutputContext struct {
	Vars      map[string]any
	Overwrite bool
	DryRun    bool
	FS        cli.FS

	working []string
	out     cli.Writer
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

func (c *OutputContext) Do(ctx context.Context, gens ...Generator) error {
	for _, g := range gens {
		err := g.Generate(ctx, c)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *OutputContext) Exists(name string) bool {
	_, err := c.Stat(name)
	return err == nil || !errors.Is(err, fs.ErrNotExist)
}

func (c *OutputContext) File(name string) string {
	if strings.HasPrefix(name, "/") {
		return name
	}
	return filepath.Join(c.WorkDir(), c.expandName(name))
}

// WorkDir is the path to the working directory
func (c *OutputContext) WorkDir() string {
	return filepath.Clean(filepath.Join(c.working...))
}

func (c *OutputContext) PushDir(name string) error {
	c.working = append(c.working, name)
	return nil
}

func (c *OutputContext) PopDir() error {
	c.working = c.working[0 : len(c.working)-1]
	if len(c.working) == 0 {
		return fmt.Errorf("cannot pop dir")
	}
	return nil
}

func (c *OutputContext) SetData(name string, value any) {
	c.Vars[name] = value
}

func (c *OutputContext) error(file string) {
	c.trace("error", file)
}

func (c *OutputContext) create(file string) {
	c.trace("create", file)
}

func (c *OutputContext) identical(file string) {
	c.trace("identical", file)
}

func (c *OutputContext) overwrite(file string) {
	c.trace("overwrite", file)
}

func (c *OutputContext) trace(category string, file string) {
	color, ok := colors[category]
	if !ok {
		color = cli.Default
	}
	out := c.out
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

func (c *OutputContext) reportChange(original []byte, name string, created bool) {
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

func (c *OutputContext) expandName(name string) string {
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

func (c *OutputContext) Stat(name string) (fs.FileInfo, error) {
	return c.actualFS().Stat(c.File(name))
}

func (c *OutputContext) Open(name string) (fs.File, error) {
	return c.actualFS().Open(c.File(name))
}

func (c *OutputContext) OpenContext(ctx context.Context, name string) (fs.File, error) {
	return c.actualFS().OpenContext(ctx, c.File(name))
}

func (c *OutputContext) Chmod(name string, mode fs.FileMode) error {
	name, err := c.pathEnsure(name)
	if err != nil {
		return err
	}
	return c.actualFS().Chmod(name, mode)
}

func (c *OutputContext) Chown(name string, uid, gid int) error {
	name, err := c.pathEnsure(name)
	if err != nil {
		return err
	}
	return c.actualFS().Chown(name, uid, gid)
}

func (c *OutputContext) Create(name string) (fs.File, error) {
	name, err := c.pathEnsure(name)
	if err != nil {
		return nil, err
	}
	return c.actualFS().Create(name)
}

func (c *OutputContext) Mkdir(name string, perm fs.FileMode) error {
	name, err := c.pathEnsure(name)
	if err != nil {
		return err
	}
	return c.actualFS().Mkdir(name, perm)
}

func (c *OutputContext) MkdirAll(path string, perm fs.FileMode) error {
	return c.actualFS().MkdirAll(c.File(path), perm)
}

func (c *OutputContext) OpenFile(name string, flag int, perm fs.FileMode) (fs.File, error) {
	name = c.File(name)
	return c.actualFS().OpenFile(name, flag, perm)
}

func (c *OutputContext) Remove(name string) error {
	name = c.File(name)
	return c.actualFS().Remove(name)
}

func (c *OutputContext) RemoveAll(path string) error {
	return c.actualFS().RemoveAll(c.File(path))
}

func (c *OutputContext) Chtimes(name string, atime time.Time, mtime time.Time) error {
	name, err := c.pathEnsure(name)
	if err != nil {
		return err
	}
	return c.actualFS().Chtimes(name, atime, mtime)
}

func (c *OutputContext) Rename(oldpath, newpath string) error {
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

func (c *OutputContext) pathEnsure(name string) (string, error) {
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

func (c *OutputContext) actualFS() cli.FS {
	return c.FS
}

var (
	_ cli.FS = (*OutputContext)(nil)
)
