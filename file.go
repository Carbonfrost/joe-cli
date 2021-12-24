package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
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

// fileExists tests whether file exists or all files in the file set exist
type fileExists interface {
	Exists() bool
}

// fileStat provides Stat() for File
type fileStat interface {
	Stat() (fs.FileInfo, error)
}

// File provides a value that can be used to represent a file path in flags or arguments.
type File struct {
	// Name is the name of the file
	Name string

	// FS specifies the file system that is used for the file.  If not specified, it provides a
	// default file system based upon the os package, which has the additional behavior that it treats the name "-"
	// specially as a file that reads and seeks from Stdin and writes to Stdout,
	FS fs.FS
}

// FileSet provides a list of files and/or directories and whether the scope of the
// file set is recursive
type FileSet struct {
	Recursive bool

	// Files provides the files named in the file set.  These can be files or
	// directories
	Files      []string
	FS         fs.FS
	initialSet bool
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

const (
	readWriteMask = os.O_RDONLY | os.O_WRONLY | os.O_RDWR
)

func newDefaultFS(in io.Reader, out io.Writer) *defaultFS {
	return &defaultFS{&stdFile{in, out}}
}

func (f *File) Set(arg string) error {
	f.Name = arg
	return nil
}

func (f *File) String() string {
	return f.Name
}

func (f *File) Ext() string {
	return filepath.Ext(f.Name)
}

func (f *File) Dir() string {
	return filepath.Dir(f.Name)
}

// Open the file
func (f *File) Open() (fs.File, error) {
	return f.actualFS().Open(f.Name)
}

// OpenFile will open the file using the specified flags and permissions
func (f *File) OpenFile(flag int, perm os.FileMode) (*os.File, error) {
	return f.actualFS().OpenFile(f.Name, flag, perm)
}

// Create the file
func (f *File) Create() (*os.File, error) {
	return f.actualFS().OpenFile(f.Name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

// Exists tests whether the file exists
func (f *File) Exists() bool {
	_, err := f.Stat()
	return err == nil || !fileNotExists(err)
}

// Stat obtains information about the file
func (f *File) Stat() (fs.FileInfo, error) {
	return f.actualFS().Stat(f.Name)
}

// Walk walks the file tree, calling fn for each file or directory in the tree, including the root.
func (f *File) Walk(fn fs.WalkDirFunc) error {
	return walkFile(f.actualFS(), f.Name, fn)
}

func (f *File) actualFS() fsExtension {
	if f.FS == nil {
		return newDefaultFS(os.Stdin, os.Stdout)
	}
	return wrapFS(f.FS)
}

func (f *FileSet) Set(arg string) error {
	if f.initialSet {
		f.Files = []string{}
		f.initialSet = true
	}

	f.Files = append(f.Files, arg)
	return nil
}

func (f *FileSet) String() string {
	return Join(f.Files)
}

// Exists tests whether all files in the set exist
func (f *FileSet) Exists() bool {
	ff := f.actualFS()
	for _, file := range f.Files {
		_, err := (&File{file, ff}).Stat()
		if fileNotExists(err) {
			return false
		}
	}
	return true
}

// Do will invoke the given function on each file in the set.  If recursion is
// enabled, it will recurse directories and process on each file encountered.
func (f *FileSet) Do(fn func(*File, error) error) error {
	ff := f.actualFS()
	if f.Recursive {
		for _, file := range f.Files {
			err := walkFile(ff, file, func(path string, _ fs.DirEntry, walkErr error) error {
				return fn(&File{path, ff}, walkErr)
			})
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, file := range f.Files {
		if err := fn(&File{file, ff}, nil); err != nil {
			return err
		}
	}
	return nil
}

func (f *FileSet) actualFS() fsExtension {
	if f.FS == nil {
		// Handling of - does not apply to file set
		return &defaultFS{}
	}
	return wrapFS(f.FS)
}

func (d defaultFS) Open(name string) (fs.File, error) {
	if name == "-" && d.std != nil {
		return d.std, nil
	}
	return os.Open(name)
}

func (d defaultFS) Stat(name string) (fs.FileInfo, error) {
	if name == "-" && d.std != nil {
		return d.std.Stat()
	}
	return os.Stat(name)
}

func (d defaultFS) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	if name == "-" && d.std != nil {
		switch flag & readWriteMask {
		case os.O_RDONLY:
			return d.std.in.(*os.File), nil
		case os.O_RDWR:
			if (flag & (os.O_APPEND | os.O_CREATE)) > 0 {
				return d.std.out.(*os.File), nil
			}
			return nil, errors.New("open not supported: O_RDWR must be specified with O_APPEND or O_CREATE")
		case os.O_WRONLY:
			return d.std.out.(*os.File), nil
		}
	}
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

func fileNotExists(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}

func wrapFS(f fs.FS) fsExtension {
	if ext, ok := f.(fsExtension); ok {
		return ext
	}
	return fsExtensionWrapper{f}
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

func walkFile(ff fsExtension, name string, fn fs.WalkDirFunc) error {
	return fs.WalkDir(ff, name, fn)
}

var (
	_ fileExtension = &stdFile{}
	_ fsExtension   = (*defaultFS)(nil)
	_ flag.Value    = (*File)(nil)
	_ flag.Value    = (*FileSet)(nil)
	_ fileExists    = (*File)(nil)
	_ fileExists    = (*FileSet)(nil)
	_ fileStat      = (*File)(nil)
)
