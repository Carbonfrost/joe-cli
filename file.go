// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cli

import (
	"bufio"
	"bytes"
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

	"golang.org/x/term"
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
	OpenContextFS
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
}

type OpenContextFS interface {
	fs.FS
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
// file set is recursive.
// A FileSet can also be read-in from a reader using the SetData
// method. The format is a list of file names, with blank lines
// or lines starting with comment characters being skipped.
// Comment characters are # and ;.
type FileSet struct {
	// Recursive determines whether iteration over the file set by
	// Do or All recursively opens directories
	Recursive bool

	// Inplace determines whether the FileInput obtained from Input writes its
	// output back to the corresponding input file rather than to standard
	// output.  See FileInput.Output for details.
	Inplace bool

	// BackupSuffix, when set together with Inplace, causes each input file to
	// be copied to a backup file whose name is the input file name with this
	// suffix appended before it is overwritten.  See FileInput.Output.
	BackupSuffix string

	// Files provides the files named in the file set.  These can be files or
	// directories
	Files []string

	// FS is the file system used to open files
	FS fs.FS

	// Globber specifies a glob function to apply when enumerating files.  When
	// set, each name in Files is expanded by calling Globber, and iteration
	// proceeds over the matched names.  Recursion is applied after globbing, so
	// directories matched by the glob are recursed into when Recursive is set.
	// When nil, names in Files are used as-is.
	Globber func(string) ([]string, error)
}

type stdFile struct {
	in      io.Reader
	out     io.Writer
	inCache *bytes.Buffer
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

// FileInput provides a controller for successively reading the files
// enumerated from a FileSet.  It is obtained from FileSet.Input.
//
// The files are enumerated by one of the scanning methods, Readers or
// Contents.  As iteration proceeds, the state of the FileInput reflects the
// file currently being processed; use Filename, File, and Err to obtain
// information about it.  Only one scanning method can be used per FileInput;
// calling a second one panics.
//
// When the file set is empty, stdin is implicitly used as the only file
// unless stdin is connected to a TTY.  When the file set names a directory,
// it is an error to read from it unless the file set is also Recursive.
//
// A FileInput also provides an output controller via Output (and the Print,
// Printf, and Println helpers).  By default the output is standard output;
// when the file set is Inplace, the output is written back to the file
// currently being processed.  Use SetInplace and SetBackupSuffix to configure
// this behavior; FileSet.Input copies these settings from the FileSet.
type FileInput struct {
	state    fileInputState
	current  *File
	err      error
	method   string
	advanced bool

	fs           fs.FS
	inplace      bool
	backupSuffix string
	out          io.Writer
}

type fileEntry struct {
	file *File
	err  error
}

type stdinReaderFS interface {
	stdin() io.Reader
}

type stdoutWriterFS interface {
	stdout() io.Writer
}

var errIsDirectory = errors.New("is a directory")

// Readers enumerates each of the files in the fileset, producing a reader
// for each one.
func (fi *FileInput) Readers() iter.Seq2[io.Reader, *FileInput] {
	fi.useMethod("Readers")
	return func(yield func(io.Reader, *FileInput) bool) {
		fi.drive(func(fi *FileInput) bool {
			var r io.Reader
			if fi.err == nil {
				r, fi.err = fi.open()
			}
			return yield(r, fi)
		})
	}
}

// Contents enumerates each of the files in the fileset, producing its contents
// in bytes.
func (fi *FileInput) Contents() iter.Seq2[[]byte, *FileInput] {
	fi.useMethod("Contents")
	return func(yield func([]byte, *FileInput) bool) {
		fi.drive(func(fi *FileInput) bool {
			var data []byte
			if fi.err == nil {
				var r io.Reader
				if r, fi.err = fi.open(); fi.err == nil {
					data, fi.err = io.ReadAll(r)
					fi.closeReader(r)
				}
			}
			return yield(data, fi)
		})
	}
}

// Filename is the name of the file currently being processed.
func (fi *FileInput) Filename() string {
	if fi.current == nil {
		return ""
	}
	return fi.current.Name
}

// File is the file currently being processed.
func (fi *FileInput) File() *File {
	return fi.current
}

// Output obtains the writer used for the output of the file currently being
// processed.  By default this is standard output.  When SetInplace has been
// used, it instead writes to the file currently being processed: the file is
// overwritten on the first write to the output.  When SetBackupSuffix has also
// been used, the input file is first copied to a backup file whose name is the
// input file name with the suffix appended.
//
// The writer is bound to the current file; as iteration advances to the next
// file, a fresh writer is returned that targets that file, and the previous
// file's output is flushed and closed.
func (fi *FileInput) Output() io.Writer {
	if fi.out == nil {
		fi.out = fi.newOutput()
	}
	return fi.out
}

// SetInplace controls whether Output writes back to the file currently being
// processed instead of standard output.
func (fi *FileInput) SetInplace(v bool) {
	fi.inplace = v
}

// SetBackupSuffix sets the suffix used to name the backup copy made of each
// input file before it is overwritten when writing output in place.  When the
// suffix is empty, no backup is made.  The suffix has no effect unless the
// input is also in place; see SetInplace.
func (fi *FileInput) SetBackupSuffix(s string) {
	fi.backupSuffix = s
}

// Print formats using the default formats for its operands and writes to
// Output.  It corresponds to fmt.Fprint(fi.Output(), a...).
func (fi *FileInput) Print(a ...any) (int, error) {
	return fmt.Fprint(fi.Output(), a...)
}

// Printf formats according to a format specifier and writes to Output.  It
// corresponds to fmt.Fprintf(fi.Output(), format, a...).
func (fi *FileInput) Printf(format string, a ...any) (int, error) {
	return fmt.Fprintf(fi.Output(), format, a...)
}

// Println formats using the default formats for its operands and writes to
// Output.  It corresponds to fmt.Fprintln(fi.Output(), a...).
func (fi *FileInput) Println(a ...any) (int, error) {
	return fmt.Fprintln(fi.Output(), a...)
}

func (fi *FileInput) newOutput() io.Writer {
	if fi.inplace && fi.current != nil {
		return &inplaceWriter{
			file:   fi.current,
			suffix: fi.backupSuffix,
		}
	}
	if out := stdoutOf(actualFS(fi.fs)); out != nil {
		return out
	}
	return os.Stdout
}

// closeOutput flushes and closes the output writer for the current file, if
// one was created.  Only the in-place writer is closed; the standard output
// writer must be left open because it is shared.
func (fi *FileInput) closeOutput() {
	if o, ok := fi.out.(*inplaceWriter); ok {
		_ = o.Close()
	}
	fi.out = nil
}

// Err contains the error reading a file or scanning its input.  The error
// io.EOF won't be returned.  If an error is returned, you must call NextFile
// to clear it in order to proceed to handling the next file within the next
// iteration; otherwise, the iteration will stop.
func (fi *FileInput) Err() error {
	if errors.Is(fi.err, io.EOF) {
		return nil
	}
	return fi.err
}

// NextFile moves to the next file in the fileset.  If Err is set, then the
// error is cleared.  It reports whether there is a further file to process.
func (fi *FileInput) NextFile() bool {
	return fi.state.nextFile(fi)
}

func (fi *FileInput) drive(yield func(*FileInput) bool) {
	fi.state.drive(fi, yield)
}

func (fi *FileInput) open() (io.Reader, error) {
	f := fi.current

	if info, err := f.Stat(); err == nil && info != nil && info.IsDir() {
		return nil, &fs.PathError{Op: "read", Path: f.Name, Err: errIsDirectory}
	}
	return f.Open()
}

func (fi *FileInput) closeReader(r io.Reader) {
	if fi.current != nil && fi.current.Name == "-" {
		// Don't close stdin because it might be shared
		return
	}
	if c, ok := r.(io.Closer); ok {
		_ = c.Close()
	}
}

func (fi *FileInput) useMethod(name string) {
	if fi.method != "" {
		panic("cli: FileInput." + name + " called after FileInput." + fi.method)
	}
	fi.method = name
}

type fileInputState interface {
	drive(fi *FileInput, yield func(*FileInput) bool)
	nextFile(fi *FileInput) bool
}

type walkerFileInputState struct {
	source iter.Seq[fileEntry]
	next   func() (fileEntry, bool)
	cur    fileEntry
	ok     bool
}

type cachedFileInputState struct {
	entries []fileEntry
	index   int
}

func (s *walkerFileInputState) drive(fi *FileInput, yield func(*FileInput) bool) {
	next, stop := iter.Pull(s.source)
	defer stop()
	defer fi.closeOutput()
	s.next = next

	s.cur, s.ok = next()
	for s.ok {
		fi.closeOutput()
		fi.current = s.cur.file
		fi.err = s.cur.err
		fi.advanced = false

		if !yield(fi) {
			return
		}

		if fi.Err() != nil && !fi.advanced {
			return
		}
		if !fi.advanced {
			s.cur, s.ok = next()
		}
	}
}

func (s *walkerFileInputState) nextFile(fi *FileInput) bool {
	fi.err = nil
	fi.advanced = true
	if s.next == nil {
		return false
	}
	s.cur, s.ok = s.next()
	return s.ok
}

func (s *cachedFileInputState) drive(fi *FileInput, yield func(*FileInput) bool) {
	defer fi.closeOutput()
	for s.index < len(s.entries) {
		fi.closeOutput()
		e := s.entries[s.index]
		fi.current = e.file
		fi.err = e.err
		fi.advanced = false

		if !yield(fi) {
			return
		}

		if fi.Err() != nil && !fi.advanced {
			return
		}
		if !fi.advanced {
			s.index++
		}
	}
}

func (s *cachedFileInputState) nextFile(fi *FileInput) bool {
	fi.err = nil
	s.index++
	fi.advanced = true
	return s.index < len(s.entries)
}

func stdinOf(ff FS) io.Reader {
	if s, ok := ff.(stdinReaderFS); ok {
		return s.stdin()
	}
	return nil
}

func stdoutOf(ff FS) io.Writer {
	if s, ok := ff.(stdoutWriterFS); ok {
		return s.stdout()
	}
	return nil
}

func isTerminalReader(r io.Reader) bool {
	if f, ok := r.(interface{ Fd() uintptr }); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

func (d defaultFS) stdin() io.Reader {
	if d.std != nil {
		return d.std.in
	}
	return nil
}

func (d defaultFS) stdout() io.Writer {
	if d.std != nil {
		return d.std.out
	}
	return nil
}

var (
	_ stdinReaderFS  = defaultFS{}
	_ stdoutWriterFS = defaultFS{}
)

// NewFS wraps the given FS for use as a CLI file system. If the argument
// is nil, the result will also be nil
func NewFS(f fs.FS) FS {
	if f == nil {
		return nil
	}
	return wrapFS(f)
}

// NewSysFS wraps the given FS with the semantics for the special
// file named with a dash.  Typically, the file named by dash
// reads from stdin and writes to stdout.
func NewSysFS(base FS, in io.Reader, out io.Writer) FS {
	return &defaultFS{
		FS:  base,
		std: &stdFile{in: in, out: out},
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

// Base obtains the basename of the file
func (f *File) Base() string {
	return filepath.Base(f.Name)
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
	return actualFS(f.FS).Open(f.Name)
}

// OpenContext is used to open the file with the given context
func (f *File) OpenContext(c context.Context) (fs.File, error) {
	return actualFS(f.FS).OpenContext(c, f.Name)
}

// OpenFile will open the file using the specified flags and permissions
func (f *File) OpenFile(flag int, perm os.FileMode) (fs.File, error) {
	return actualFS(f.FS).OpenFile(f.Name, flag, perm)
}

// Create the file
func (f *File) Create() (fs.File, error) {
	return actualFS(f.FS).Create(f.Name)
}

// Chmod to change mode
func (f *File) Chmod(mode fs.FileMode) error {
	return actualFS(f.FS).Chmod(f.Name, mode)
}

// Chown to change owner
func (f *File) Chown(uid int, gid int) error {
	return actualFS(f.FS).Chown(f.Name, uid, gid)
}

// Chtimes to change times
func (f *File) Chtimes(atime, mtime time.Time) error {
	return actualFS(f.FS).Chtimes(f.Name, atime, mtime)
}

// Rename file
func (f *File) Rename(newpath string) error {
	return actualFS(f.FS).Rename(f.Name, newpath)
}

// Remove file
func (f *File) Remove() error {
	return actualFS(f.FS).Remove(f.Name)
}

// RemoveAll to remove file and all ancestors
func (f *File) RemoveAll() error {
	return actualFS(f.FS).RemoveAll(f.Name)
}

// Mkdir creates a directory
func (f *File) Mkdir(mode fs.FileMode) error {
	return actualFS(f.FS).Mkdir(f.Name, mode)
}

// MkdirAll creates a directory and all ancestors
func (f *File) MkdirAll(mode fs.FileMode) error {
	return actualFS(f.FS).MkdirAll(f.Name, mode)
}

// Exists tests whether the file exists
func (f *File) Exists() bool {
	_, err := f.Stat()
	return err == nil || !fileNotExists(err)
}

// Stat obtains information about the file
func (f *File) Stat() (fs.FileInfo, error) {
	return actualFS(f.FS).Stat(f.Name)
}

// Walk walks the file tree, calling fn for each file or directory in the tree, including the root.
func (f *File) Walk(fn fs.WalkDirFunc) error {
	return walkFile(actualFS(f.FS), f.Name, fn)
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

// OpenReader gets the reader for the file
func (f *File) OpenReader() io.Reader {
	r, err := f.Open()
	if err != nil {
		return errOnFirstReader{err}
	}
	return r
}

type errOnFirstReader struct {
	err error
}

func (e errOnFirstReader) Read([]byte) (int, error) {
	return -1, e.err
}

// CreateWriter gets the writer for the file
func (f *File) CreateWriter() io.Writer {
	r, err := f.Create()
	if err != nil {
		return errOnFirstWriter{err}
	}
	return r.(io.Writer)
}

type errOnFirstWriter struct {
	err error
}

func (e errOnFirstWriter) Write([]byte) (int, error) {
	return -1, e.err
}

// inplaceWriter writes back to file, deferring the (destructive) opening of
// the file until the first write.  When suffix is set, the file is copied to a
// backup before it is overwritten.
type inplaceWriter struct {
	file    *File
	suffix  string
	w       io.Writer
	started bool
}

func (o *inplaceWriter) Write(p []byte) (int, error) {
	if !o.started {
		o.started = true
		o.w = o.begin()
	}
	return o.w.Write(p)
}

func (o *inplaceWriter) begin() io.Writer {
	if o.suffix != "" {
		if err := backupFile(o.file, o.file.Name+o.suffix); err != nil {
			return errOnFirstWriter{err}
		}
	}
	f, err := o.file.Create()
	if err != nil {
		return errOnFirstWriter{err}
	}
	return f.(io.Writer)
}

func (o *inplaceWriter) Close() error {
	if c, ok := o.w.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

// backupFile copies the contents of src to a new file named dest.
func backupFile(src *File, dest string) error {
	in, err := src.Open()
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := (&File{Name: dest, FS: src.FS}).Create()
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out.(io.Writer), in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

// Set argument value; can call repeatedly
func (f *FileSet) Set(arg string) error {
	if f.Files == nil {
		f.Files = []string{}
	}

	f.Files = append(f.Files, arg)
	return nil
}

// SetReader reads in a list of paths from a reader.
// Blank lines and comment lines (using ; or #) will be
// ignored. Whitespace is trimmed.
func (f *FileSet) SetReader(in io.Reader) error {
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		f.Files = append(f.Files, line)
	}
	return scanner.Err()
}

func (f *FileSet) String() string {
	return Join(f.Files)
}

// Exists tests whether all files in the set exist
func (f *FileSet) Exists() bool {
	ff := actualFS(f.FS)
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

// Do will invoke the given function on each file in the set.  If a Globber is
// specified, each name is first expanded into its matches.  If recursion is
// enabled, it will recurse directories and process on each file encountered.
// Recursion is applied after globbing.
func (f *FileSet) Do(fn func(*File, error) error) error {
	ff := actualFS(f.FS)
	files, err := f.globbed()
	if err != nil {
		return err
	}

	if f.Recursive {
		for _, file := range files {
			err := walkFile(ff, file, func(path string, _ fs.DirEntry, walkErr error) error {
				return fn(&File{path, ff}, walkErr)
			})
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, file := range files {
		if err := fn(&File{file, ff}, nil); err != nil {
			return err
		}
	}
	return nil
}

// globbed expands the names in Files using the Globber, if set.  When no
// Globber is present, the names are returned as-is.
func (f *FileSet) globbed() ([]string, error) {
	if f.Globber == nil {
		return f.Files, nil
	}

	files := make([]string, 0, len(f.Files))
	for _, file := range f.Files {
		matches, err := f.Globber(file)
		if err != nil {
			return nil, err
		}
		files = append(files, matches...)
	}
	return files, nil
}

// NewCounter obtains the arg counter for file sets, which is implied to be TakeUntilNextFlag
func (f *FileSet) NewCounter() ArgCounter {
	return ArgCount(TakeUntilNextFlag)
}

// Initializer obtains the initializer for the FileSet, which is used to setup the file system used
func (f *FileSet) Initializer() Action {
	return ActionFunc(f.setupOptionRequireFS)
}

func (f *FileSet) setRecursive(b bool) error {
	f.Recursive = b
	return nil
}

func (f *FileSet) setInplace(b bool) error {
	f.Inplace = b
	return nil
}

func (f *FileSet) setBackupSuffix(s string) error {
	f.BackupSuffix = s
	return nil
}

// RecursiveFlag obtains a conventions-based flag for making the file set recursive.
func (f *FileSet) RecursiveFlag() Prototype {
	return Prototype{
		Name:     "recursive",
		HelpText: "Include files and directories recursively",
		Uses:     bind(f.setRecursive),
	}
}

// InplaceFlag obtains a conventions-based flag for editing files in place.  It
// sets Inplace so that the output of the file set's Input is written back to
// each input file rather than to standard output.
func (f *FileSet) InplaceFlag() Prototype {
	return Prototype{
		Name:     "in-place",
		Aliases:  []string{"i"},
		HelpText: "Edit files in place instead of writing to standard output",
		Uses:     bind(f.setInplace),
	}
}

// BackupSuffixFlag obtains a conventions-based flag for setting the suffix used
// to back up each file before it is edited in place.  It sets BackupSuffix,
// which takes effect when editing in place; see InplaceFlag.
func (f *FileSet) BackupSuffixFlag() Prototype {
	return Prototype{
		Name:     "suffix",
		HelpText: "Back up each file to a copy named with the given `SUFFIX` before editing in place",
		Uses:     bind(f.setBackupSuffix),
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

// Input obtains a controller for successively reading the files enumerated
// from the file set.
func (f *FileSet) Input() *FileInput {
	return &FileInput{
		state: &walkerFileInputState{
			source: f.entries(),
		},
		fs:           f.FS,
		inplace:      f.Inplace,
		backupSuffix: f.BackupSuffix,
	}
}

// CachedInput obtains a controller for successively reading file enumerated from
// the file set; however, compared to Input, the results are pre-calculated.
// Typically, the FileInput walks the hierarchy in a lazy fashion
// which may be desirable for a large number of files; however, if errors occur,
// they may be deferred which could make it more complex to handle a complete and
// consistent iteration of all files. CachedInput helps by calculating all files and
// detecting any errors upfront.
func (f *FileSet) CachedInput() (*FileInput, error) {
	e, err := f.acchedEntries()
	if err != nil {
		return nil, err
	}
	return &FileInput{
		state: &cachedFileInputState{
			entries: e,
		},
		fs:           f.FS,
		inplace:      f.Inplace,
		backupSuffix: f.BackupSuffix,
	}, nil
}

// entries provides the lazy enumerator
func (f *FileSet) entries() iter.Seq[fileEntry] {
	if f.Globber != nil {
		panic("not implemented: FileInput with globber")
	}
	return func(yield func(fileEntry) bool) {
		ff := actualFS(f.FS)

		if len(f.Files) == 0 {
			if checkStdin(ff) {
				yield(fileEntry{file: &File{"-", ff}})
			}
			return
		}

		for _, name := range f.Files {
			if f.Recursive {
				err := walkFile(ff, name, func(path string, d fs.DirEntry, walkErr error) error {
					var e fileEntry
					switch {
					case walkErr != nil:
						e = fileEntry{&File{path, ff}, walkErr}
					case d.IsDir():
						return nil
					default:
						e = fileEntry{file: &File{path, ff}}
					}
					if !yield(e) {
						return errStopWalk
					}
					return nil
				})
				if errors.Is(err, errStopWalk) {
					return
				}
				continue
			}

			if !yield(fileEntry{file: &File{name, ff}}) {
				return
			}
		}
	}
}

// acchedEntries provides the upfront enumeartor
func (f *FileSet) acchedEntries() ([]fileEntry, error) {
	ff := actualFS(f.FS)

	if len(f.Files) == 0 {
		// With no files, stdin is used implicitly unless it is a TTY.
		if checkStdin(ff) {
			return []fileEntry{{file: &File{"-", ff}}}, nil
		}
		return nil, nil
	}

	var entries []fileEntry
	gg, err := f.globbed()
	if err != nil {
		return nil, err
	}
	for _, name := range gg {
		if f.Recursive {
			_ = walkFile(ff, name, func(path string, d fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					entries = append(entries, fileEntry{&File{path, ff}, walkErr})
					return nil
				}
				if d.IsDir() {
					return nil
				}
				entries = append(entries, fileEntry{file: &File{path, ff}})
				return nil
			})
			continue
		}

		entries = append(entries, fileEntry{file: &File{name, ff}})
	}
	return entries, nil
}

func checkStdin(fsys FS) bool {
	in := stdinOf(fsys)
	return in != nil && !isTerminalReader(in)
}

func (d defaultFS) Open(name string) (fs.File, error) {
	return d.OpenContext(context.Background(), name)
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
			return d.std.in.(fs.File), nil
		case os.O_RDWR:
			if (flag & (os.O_APPEND | os.O_CREATE)) > 0 {
				return d.std.out.(fs.File), nil
			}
			return nil, errors.New("open not supported: O_RDWR must be specified with O_APPEND or O_CREATE")
		case os.O_WRONLY:
			return d.std.out.(fs.File), nil
		}
	}
	return d.FS.OpenFile(name, flag, perm)
}

func (d defaultFS) OpenContext(c context.Context, name string) (fs.File, error) {
	if name == "-" && d.std != nil {
		return d.std.copy(), nil
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
	if f, ok := s.in.(fileStat); ok {
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

func (s *stdFile) copy() *stdFile {
	if s.inCache == nil {
		var cache bytes.Buffer
		_, _ = cache.ReadFrom(s.in)
		s.inCache = &cache
	}
	return &stdFile{
		in:  bytes.NewReader(s.inCache.Bytes()),
		out: s.out,
	}
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
	return os.Stat(d.path(name))
}

func (d dirFS) Open(name string) (fs.File, error) {
	return os.Open(d.path(name))
}

func (d dirFS) Create(name string) (fs.File, error) {
	return os.Create(d.path(name))
}

func (d dirFS) Chmod(name string, mode fs.FileMode) error {
	return os.Chmod(d.path(name), mode)
}

func (d dirFS) Chtimes(name string, atime, mtime time.Time) error {
	return os.Chtimes(d.path(name), atime, mtime)
}

func (d dirFS) Chown(name string, uid int, gid int) error {
	return os.Chown(d.path(name), uid, gid)
}

func (d dirFS) Rename(oldpath, newpath string) error {
	return os.Rename(d.path(oldpath), d.path(newpath))
}

func (d dirFS) Remove(name string) error {
	return os.Remove(d.path(name))
}

func (d dirFS) RemoveAll(name string) error {
	return os.RemoveAll(d.path(name))
}

func (d dirFS) Mkdir(name string, mode fs.FileMode) error {
	return os.Mkdir(d.path(name), mode)
}

func (d dirFS) MkdirAll(name string, mode fs.FileMode) error {
	return os.MkdirAll(d.path(name), mode)
}

func (d dirFS) OpenFile(name string, flag int, perm fs.FileMode) (fs.File, error) {
	return os.OpenFile(d.path(name), flag, perm)
}

func (d dirFS) OpenContext(c context.Context, name string) (fs.File, error) {
	f, err := os.Open(d.path(name))
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
	return dirFS(d.path(name)), nil
}

func (d dirFS) path(name string) string {
	if strings.HasPrefix(name, "/") {
		return name
	}
	return path.Join(string(d), name)
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

func actualFS(f ...fs.FS) FS {
	if len(f) == 0 || f[0] == nil {
		return newDefaultFS(os.Stdin, os.Stdout)
	}
	return wrapFS(f[0])
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
