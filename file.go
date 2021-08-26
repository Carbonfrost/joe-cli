package cli

import (
	"errors"
	"io/fs"
	"os"
)

type File struct {
	v string
}

func (f *File) Set(arg string) error {
	f.v = arg
	return nil
}

func (f *File) String() string {
	return f.v
}

func (f *File) Open() (*os.File, error) {
	return os.Open(f.v)
}

func (f *File) OpenFile(flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(f.v, flag, perm)
}

func (f *File) Create() (*os.File, error) {
	return os.Create(f.v)
}

func (f *File) Name() string {
	return f.v
}

func (f *File) Exists() bool {
	_, err := os.Stat(f.v)
	return err == nil || !errors.Is(err, fs.ErrNotExist)
}

func (f *File) Stat() (fs.FileInfo, error) {
	return os.Stat(f.v)
}
