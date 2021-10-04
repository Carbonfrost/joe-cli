package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
)

type fileExtension interface {
	fs.File
	io.Seeker
	io.Closer
	io.Writer
	io.ReaderAt
	io.StringWriter
}

type fsExtension interface {
	fs.FS
	fs.StatFS
	OpenFile(string, int, os.FileMode) (*os.File, error)
}

// File provides a value that can be used to represent a file path in flags or arguments.
type File struct {
	v string

	// FS specifies the file system that is used for the file.  If not specified, it provides a
	// default file system based upon the os package, which has the additional behavior that it treats the name "-"
	// specially as a file that reads and seeks from Stdin and writes to Stdout,
	FS fs.FS
}

type stdFile struct {
	in  io.Reader
	out io.Writer
}

type defaultFS struct {
	std *stdFile
}

type fsExtensionWrapper struct {
	fs.FS
}

func newDefaultFS(in io.Reader, out io.Writer) *defaultFS {
	return &defaultFS{&stdFile{in, out}}
}

func (f *File) Set(arg string) error {
	f.v = arg
	return nil
}

func (f *File) String() string {
	return f.v
}

// Open the file
func (f *File) Open() (fs.File, error) {
	return f.actualFS().Open(f.v)
}

// OpenFile will open the file using the specified flags and permissions
func (f *File) OpenFile(flag int, perm os.FileMode) (*os.File, error) {
	return f.actualFS().OpenFile(f.Name(), flag, perm)
}

// Create the file
func (f *File) Create() (*os.File, error) {
	return f.OpenFile(os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

// Name of the file
func (f *File) Name() string {
	return f.v
}

// Exists tests whether the file exists
func (f *File) Exists() bool {
	_, err := f.Stat()
	return err == nil || !errors.Is(err, fs.ErrNotExist)
}

// Stat obtains information about the file
func (f *File) Stat() (fs.FileInfo, error) {
	return f.actualFS().Stat(f.v)
}

func (f *File) actualFS() fsExtension {
	if f.FS == nil {
		return newDefaultFS(os.Stdin, os.Stdout)
	}
	return fsExtensionWrapper{f.FS}
}

func (d defaultFS) Open(name string) (fs.File, error) {
	if name == "-" {
		return d.std, nil
	}
	return os.Open(name)
}

func (d defaultFS) Stat(name string) (fs.FileInfo, error) {
	if name == "-" {
		return d.std.Stat()
	}
	return os.Stat(name)
}

func (d defaultFS) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

func (f fsExtensionWrapper) Stat(name string) (fs.FileInfo, error) {
	if s, ok := f.FS.(fs.StatFS); ok {
		return s.Stat(name)
	}
	return nil, &fs.PathError{
		Op:   "stat",
		Path: name,
		Err:  errors.New("not supported"),
	}
}

func (f fsExtensionWrapper) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return nil, &fs.PathError{
		Op:   "open",
		Path: name,
		Err:  errors.New("not supported"),
	}
}

func (s *stdFile) Stat() (fs.FileInfo, error) {
	if f, ok := s.in.(*os.File); ok {
		return f.Stat()
	}
	return nil, nil
}

func (s *stdFile) Read(d []byte) (int, error) {
	return s.in.Read(d)
}

func (s *stdFile) Close() error {
	var err error
	if c, ok := s.in.(io.Closer); ok {
		err = c.Close()
	}
	if c, ok := s.out.(io.Closer); ok {
		if e := c.Close(); e != nil {
			err = e
		}
	}
	return err
}

func (s *stdFile) ReadAt(p []byte, off int64) (n int, err error) {
	return s.in.(io.ReaderAt).ReadAt(p, off)
}

func (s *stdFile) Seek(offset int64, whence int) (int64, error) {
	return s.in.(io.Seeker).Seek(offset, whence)
}

func (s *stdFile) WriteString(str string) (n int, err error) {
	if w, ok := s.out.(io.StringWriter); ok {
		return w.WriteString(str)
	}
	return fmt.Fprint(s.out, str)
}

func (s *stdFile) Write(p []byte) (n int, err error) {
	return s.out.Write(p)
}

func setupOptionRequireFS(c *Context) error {
	if f, ok := c.option().value().(*File); ok {
		if f.FS == nil {
			app := c.App()
			fs := app.FS

			if fs == nil {
				fs = newDefaultFS(app.Stdin, app.Stdout)
			}
			f.FS = fs
		}
	}
	return nil
}

var (
	_ fileExtension = &stdFile{}
	_ fs.FS         = &defaultFS{}
)
