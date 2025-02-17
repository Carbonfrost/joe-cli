package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

//counterfeiter:generate . FS

// FS provides a read-write file system.  In addition to the read-only behavior of io/fs.FS, it
// provides all the semantics available from the package os.  To obtain an instance for the file
// system provided by package os, use DirFS.  The io/fs.File value returned from any of the methods of this
// interface can also provide corresponding methods.  For example, because FS defines
// Remove(string), the File implementation returned from Open or OpenFile can also implement the
// method Remove().
//
// Though it implements fs.FS, implementers do not abide the restrictions for path name validations
// required by io/fs.FS.Open().  In particular, starting and trailing slashes in path names are allowed,
// which allows rooted files to be referenced.  (These are the semantics of os.Open, etc.)
type FS interface {
	fs.FS
	fs.StatFS
	Chmod(name string, mode fs.FileMode) error
	Chown(name string, uid, gid int) error
	Create(name string) (fs.File, error)
	Mkdir(name string, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	OpenFile(name string, flag int, perm fs.FileMode) (fs.File, error)
	Remove(name string) error
	RemoveAll(path string) error
	Chtimes(name string, atime time.Time, mtime time.Time) error
	Rename(oldpath, newpath string) error
	OpenContext(context.Context, string) (fs.File, error)
}

type fileExtension interface {
	fs.File
	io.Seeker
	io.Closer
	io.Writer
	io.ReaderAt
	io.StringWriter
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
	Files []string
	FS    fs.FS
}

type stdFile struct {
	in  io.Reader
	out io.Writer
}

type defaultFS struct {
	FS
	std *stdFile
}

type contextFile struct {
	*os.File
	ctx context.Context
}

type fsExtensionWrapper struct {
	fs.FS
}

type dirFS string

var errStopWalk = errors.New("stop walking")

// NewFS wraps the given FS for use as a CLI file system.
func NewFS(f fs.FS) FS {
	return wrapFS(f)
}

// NewSysFS wraps the given FS with the semantics for the special
// file named with a dash.  Typically, the file named by dash
// reads from stdin and writes to stdout.
func NewSysFS(base FS, in io.Reader, out io.Writer) FS {
	return &defaultFS{
		FS:  base,
		std: &stdFile{in, out},
	}
}

// DirFS returns a file system for files rooted at the directory dir.
func DirFS(dir string) FS {
	return dirFS(dir)
}

const (
	readWriteMask = os.O_RDONLY | os.O_WRONLY | os.O_RDWR
)

func newDefaultFS(in io.Reader, out io.Writer) FS {
	return NewSysFS(DirFS("."), in, out)
}

// Set will set the name of the file
func (f *File) Set(arg string) error {
	f.Name = arg
	return nil
}

func (f *File) String() string {
	return f.Name
}

// Ext obtains the file extension
func (f *File) Ext() string {
	return filepath.Ext(f.Name)
}

// Dir obtains the directory
func (f *File) Dir() string {
	return filepath.Dir(f.Name)
}

// Open the file
func (f *File) Open() (fs.File, error) {
	return f.actualFS().Open(f.Name)
}

// OpenContext is used to open the file with the given context
func (f *File) OpenContext(c context.Context) (fs.File, error) {
	return f.actualFS().OpenContext(c, f.Name)
}

// OpenFile will open the file using the specified flags and permissions
func (f *File) OpenFile(flag int, perm os.FileMode) (fs.File, error) {
	return f.actualFS().OpenFile(f.Name, flag, perm)
}

// Create the file
func (f *File) Create() (fs.File, error) {
	return f.actualFS().Create(f.Name)
}

// Chmod to change mode
func (f *File) Chmod(mode fs.FileMode) error {
	return f.actualFS().Chmod(f.Name, mode)
}

// Chown to change owner
func (f *File) Chown(uid int, gid int) error {
	return f.actualFS().Chown(f.Name, uid, gid)
}

// Chtimes to change times
func (f *File) Chtimes(atime, mtime time.Time) error {
	return f.actualFS().Chtimes(f.Name, atime, mtime)
}

// Rename file
func (f *File) Rename(newpath string) error {
	return f.actualFS().Rename(f.Name, newpath)
}

// Remove file
func (f *File) Remove() error {
	return f.actualFS().Remove(f.Name)
}

// RemoveAll to remove file and all ancestors
func (f *File) RemoveAll() error {
	return f.actualFS().RemoveAll(f.Name)
}

// Mkdir creates a directory
func (f *File) Mkdir(mode fs.FileMode) error {
	return f.actualFS().Mkdir(f.Name, mode)
}

// MkdirAll creates a directory and all ancestors
func (f *File) MkdirAll(mode fs.FileMode) error {
	return f.actualFS().MkdirAll(f.Name, mode)
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

// Initializer obtains the initializer for the File, which is used to setup the file system used
func (f *File) Initializer() Action {
	return ActionFunc(f.setupOptionRequireFS)
}

// Completion gets the completion for files
func (f *File) Completion() Completion {
	return FileCompletion
}

func (*File) Synopsis() string {
	return "FILE"
}

func (f *File) setupOptionRequireFS(c *Context) error {
	if f.FS == nil {
		f.FS = c.actualFS()
	}
	return nil
}

func (f *File) actualFS() FS {
	if f.FS == nil {
		return newDefaultFS(os.Stdin, os.Stdout)
	}
	return wrapFS(f.FS)
}

// Set argument value; can call repeatedly
func (f *FileSet) Set(arg string) error {
	if f.Files == nil {
		f.Files = []string{}
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

// Reset will resets the file set to empty
func (f *FileSet) Reset() {
	f.Files = nil
}

func (f *FileSet) All() iter.Seq2[*File, error] {
	return func(yield func(*File, error) bool) {
		f.Do(func(f *File, err error) error {
			ok := yield(f, err)
			if !ok {
				return errStopWalk
			}
			return nil
		})
	}
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

func (f *FileSet) actualFS() FS {
	if f.FS == nil {
		return newDefaultFS(os.Stdin, os.Stdout)
	}
	return wrapFS(f.FS)
}

// NewCounter obtains the arg counter for file sets, which is implied to be TakeUntilNextFlag
func (f *FileSet) NewCounter() ArgCounter {
	return ArgCount(TakeUntilNextFlag)
}

// Initializer obtains the initializer for the FileSet, which is used to setup the file system used
func (f *FileSet) Initializer() Action {
	return ActionFunc(f.setupOptionRequireFS)
}

// SetRecursive updates the file set Recursive field.  This is generally meant to be
// used with BindIndirect.  Never returns an error.
func (f *FileSet) SetRecursive(b bool) error {
	f.Recursive = b
	return nil
}

// RecursiveFlag obtains a conventions-based flag for making the file set recursive.
func (f *FileSet) RecursiveFlag() Prototype {
	return Prototype{
		Name:     "recursive",
		HelpText: "Include files and directories recursively",
		Setup: Setup{
			Uses: Bind(f.SetRecursive),
		},
	}
}

// Completion gets the completion for files
func (f *FileSet) Completion() Completion {
	return FileCompletion
}

func (*FileSet) Synopsis() string {
	return "FILES"
}

func (f *FileSet) setupOptionRequireFS(c *Context) error {
	if f.FS == nil {
		f.FS = c.FS
	}
	return nil
}

func (d defaultFS) Open(name string) (fs.File, error) {
	return d.OpenContext(context.TODO(), name)
}

// Force consolidation of Create via OpenFile (can't use the embedded value
// directly)

func (d defaultFS) Create(name string) (fs.File, error) {
	return d.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

func (d defaultFS) Stat(name string) (fs.FileInfo, error) {
	if name == "-" && d.std != nil {
		return d.std.Stat()
	}
	return d.FS.Stat(name)
}

func (d defaultFS) OpenFile(name string, flag int, perm os.FileMode) (fs.File, error) {
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
	return d.FS.OpenFile(name, flag, perm)
}

func (d defaultFS) OpenContext(c context.Context, name string) (fs.File, error) {
	if name == "-" && d.std != nil {
		return d.std, nil
	}
	return d.FS.OpenContext(c, name)
}

func (f fsExtensionWrapper) Create(name string) (fs.File, error) {
	if s, ok := f.FS.(interface{ Create(string) (fs.File, error) }); ok {
		return s.Create(name)
	}
	return f.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

func (f fsExtensionWrapper) Chmod(name string, mode fs.FileMode) error {
	if s, ok := f.FS.(interface {
		Chmod(string, fs.FileMode) error
	}); ok {
		return s.Chmod(name, mode)
	}
	return &fs.PathError{
		Op:   "chmod",
		Path: name,
		Err:  errors.New("not supported"),
	}
}

func (f fsExtensionWrapper) Chown(name string, uid int, gid int) error {
	if s, ok := f.FS.(interface {
		Chown(string, int, int) error
	}); ok {
		return s.Chown(name, uid, gid)
	}
	return &fs.PathError{
		Op:   "chown",
		Path: name,
		Err:  errors.New("not supported"),
	}
}

func (f fsExtensionWrapper) Chtimes(name string, atime, mtime time.Time) error {
	if s, ok := f.FS.(interface {
		Chtimes(string, time.Time, time.Time) error
	}); ok {
		return s.Chtimes(name, atime, mtime)
	}
	return &fs.PathError{
		Op:   "chtimes",
		Path: name,
		Err:  errors.New("not supported"),
	}
}

func (f fsExtensionWrapper) Rename(oldpath, newpath string) error {
	if s, ok := f.FS.(interface{ Rename(string, string) error }); ok {
		return s.Rename(oldpath, newpath)
	}
	return &fs.PathError{
		Op:   "rename",
		Path: oldpath,
		Err:  errors.New("not supported"),
	}
}

func (f fsExtensionWrapper) Remove(name string) error {
	if s, ok := f.FS.(interface{ Remove(string) error }); ok {
		return s.Remove(name)
	}
	return &fs.PathError{
		Op:   "remove",
		Path: name,
		Err:  errors.New("not supported"),
	}
}

func (f fsExtensionWrapper) RemoveAll(name string) error {
	if s, ok := f.FS.(interface{ RemoveAll(string) error }); ok {
		return s.RemoveAll(name)
	}
	return &fs.PathError{
		Op:   "remove",
		Path: name,
		Err:  errors.New("not supported"),
	}
}

func (f fsExtensionWrapper) Mkdir(name string, mode fs.FileMode) error {
	if s, ok := f.FS.(interface {
		Mkdir(string, fs.FileMode) error
	}); ok {
		return s.Mkdir(name, mode)
	}
	return &fs.PathError{
		Op:   "mkdir",
		Path: name,
		Err:  errors.New("not supported"),
	}
}

func (f fsExtensionWrapper) MkdirAll(name string, mode fs.FileMode) error {
	if s, ok := f.FS.(interface {
		MkdirAll(string, fs.FileMode) error
	}); ok {
		return s.MkdirAll(name, mode)
	}
	return &fs.PathError{
		Op:   "mkdir",
		Path: name,
		Err:  errors.New("not supported"),
	}
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

func (f fsExtensionWrapper) OpenContext(c context.Context, name string) (fs.File, error) {
	if s, ok := f.FS.(interface {
		OpenContext(context.Context, string) (fs.File, error)
	}); ok {
		return s.OpenContext(c, name)
	}
	return f.FS.Open(name)
}

func (f fsExtensionWrapper) OpenFile(name string, flag int, perm fs.FileMode) (fs.File, error) {
	if s, ok := f.FS.(interface {
		OpenFile(string, int, fs.FileMode) (fs.File, error)
	}); ok {
		return s.OpenFile(name, flag, perm)
	}
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

func (c *contextFile) Read(p []byte) (n int, err error) {
	if err = c.ctx.Err(); err != nil {
		return
	}
	if n, err = c.File.Read(p); err != nil {
		return
	}
	err = c.ctx.Err()
	return
}

func (c *contextFile) Write(p []byte) (n int, err error) {
	if err = c.ctx.Err(); err != nil {
		return
	}
	if n, err = c.File.Write(p); err != nil {
		return
	}
	err = c.ctx.Err()
	return
}

func (d dirFS) Stat(name string) (fs.FileInfo, error) {
	full, err := d.path(name)
	if err != nil {
		return nil, err
	}
	return os.Stat(full)
}

func (d dirFS) Open(name string) (fs.File, error) {
	full, err := d.path(name)
	if err != nil {
		return nil, err
	}
	return os.Open(full)
}

func (d dirFS) Create(name string) (fs.File, error) {
	full, err := d.path(name)
	if err != nil {
		return nil, err
	}
	return os.Create(full)
}

func (d dirFS) Chmod(name string, mode fs.FileMode) error {
	full, err := d.path(name)
	if err != nil {
		return err
	}
	return os.Chmod(full, mode)
}

func (d dirFS) Chtimes(name string, atime, mtime time.Time) error {
	full, err := d.path(name)
	if err != nil {
		return err
	}
	return os.Chtimes(full, atime, mtime)
}

func (d dirFS) Chown(name string, uid int, gid int) error {
	full, err := d.path(name)
	if err != nil {
		return err
	}
	return os.Chown(full, uid, gid)
}

func (d dirFS) Rename(oldpath, newpath string) error {
	old, err := d.path(oldpath)
	if err != nil {
		return err
	}
	new, err := d.path(newpath)
	if err != nil {
		return err
	}
	return os.Rename(old, new)
}

func (d dirFS) Remove(name string) error {
	full, err := d.path(name)
	if err != nil {
		return err
	}
	return os.Remove(full)
}

func (d dirFS) RemoveAll(name string) error {
	full, err := d.path(name)
	if err != nil {
		return err
	}
	return os.RemoveAll(full)
}

func (d dirFS) Mkdir(name string, mode fs.FileMode) error {
	full, err := d.path(name)
	if err != nil {
		return err
	}
	return os.Mkdir(full, mode)
}

func (d dirFS) MkdirAll(name string, mode fs.FileMode) error {
	full, err := d.path(name)
	if err != nil {
		return err
	}
	return os.MkdirAll(full, mode)
}

func (d dirFS) OpenFile(name string, flag int, perm fs.FileMode) (fs.File, error) {
	full, err := d.path(name)
	if err != nil {
		return nil, err
	}
	return os.OpenFile(full, flag, perm)
}

func (d dirFS) OpenContext(c context.Context, name string) (fs.File, error) {
	full, err := d.path(name)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(full)
	if err != nil {
		return nil, err
	}
	return wrapContextFile(c, f), nil
}

func (d dirFS) Sub(name string) (fs.FS, error) {
	// Implementation of Sub is to keep read-write semantics
	if name == "." {
		return d, nil
	}
	p, err := d.path(name)
	if err != nil {
		return nil, err
	}
	return dirFS(p), nil
}

func (d dirFS) path(name string) (string, error) {
	if strings.HasPrefix(name, "/") {
		return name, nil
	}
	return path.Join(string(d), name), nil
}

func fileNotExists(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}

func wrapFS(f fs.FS) FS {
	if ext, ok := f.(FS); ok {
		return ext
	}
	return fsExtensionWrapper{f}
}

func wrapContextFile(ctx context.Context, f *os.File) fs.File {
	if ctx == nil {
		return f
	}
	if deadline, ok := ctx.Deadline(); ok {
		f.SetWriteDeadline(deadline)
		f.SetReadDeadline(deadline)
	}
	return &contextFile{f, ctx}
}

func walkFile(ff FS, name string, fn fs.WalkDirFunc) error {
	return fs.WalkDir(ff, name, fn)
}

func ignoreBlankPathError(err error) error {
	if p, ok := err.(*fs.PathError); ok {
		if p.Path == "" {
			return nil
		}
	}
	return err
}

var (
	_ fileExtension    = &stdFile{}
	_ fileExtension    = &contextFile{}
	_ flag.Value       = (*File)(nil)
	_ flag.Value       = (*FileSet)(nil)
	_ valueInitializer = (*File)(nil)
	_ valueInitializer = (*FileSet)(nil)
	_ fileExists       = (*File)(nil)
	_ fileExists       = (*FileSet)(nil)
	_ fileStat         = (*File)(nil)
	_ FS               = dirFS("")
	_ fs.SubFS         = dirFS("")
)
