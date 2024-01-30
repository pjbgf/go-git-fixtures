// Package embedfs provides a billy filesystem for Go's embed fs.
package embedfs

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/go-git/go-billy/v5"
)

// Basic
// Dir

const separator = filepath.Separator

// EmbedFS exposes Go embed FS as Billy Filesystem
// New returns a Billy filesystem that implements billy.Basic and billy.Dir.
type EmbedFS struct {
	fs embed.FS
}

func New(data embed.FS) *EmbedFS {
	return &EmbedFS{fs: data}
}

func (_ *EmbedFS) Create(filename string) (billy.File, error) {
	return nil, fmt.Errorf("cannot create file: %w", billy.ErrReadOnly)
}

func (e *EmbedFS) Open(filename string) (billy.File, error) {
	return e.OpenFile(filename, os.O_RDONLY, 0)
}

func (e *EmbedFS) OpenFile(filename string, flag int, perm os.FileMode) (billy.File, error) {
	if flag&(os.O_CREATE|os.O_APPEND|os.O_TRUNC|os.O_RDWR|os.O_WRONLY) != 0 {
		return nil, fmt.Errorf("unsupported flag %v: %w", flag, billy.ErrReadOnly)
	}

	f, err := e.fs.Open(filename)
	if err != nil {
		return nil, err
	}

	return NewFile(f)
}

func (e *EmbedFS) Stat(filename string) (os.FileInfo, error) {
	f, err := e.fs.Open(filename)
	if err != nil {
		return nil, err
	}

	return f.Stat()
}

type ByName []os.FileInfo

func (a ByName) Len() int           { return len(a) }
func (a ByName) Less(i, j int) bool { return a[i].Name() < a[j].Name() }
func (a ByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func (e *EmbedFS) ReadDir(path string) ([]os.FileInfo, error) {
	dirEntries, err := e.fs.ReadDir(path)
	if err != nil {
		return nil, err
	}

	entries := make([]os.FileInfo, len(dirEntries))
	for _, f := range dirEntries {
		fi, _ := f.Info()
		entries = append(entries, fi)
	}

	sort.Sort(ByName(entries))

	return entries, nil
}

func (_ *EmbedFS) MkdirAll(path string, perm os.FileMode) error {
	return fmt.Errorf("cannot mkdirall: %w", billy.ErrReadOnly)
}

func (_ *EmbedFS) Rename(from, to string) error {
	return fmt.Errorf("cannot rename file: %w", billy.ErrReadOnly)
}

func (_ *EmbedFS) Remove(filename string) error {
	return fmt.Errorf("cannot remove file: %w", billy.ErrReadOnly)
}

func (_ *EmbedFS) Join(elem ...string) string {
	return filepath.Join(elem...)
}

// Capabilities implements the Capable interface.
func (_ *EmbedFS) Capabilities() billy.Capability {
	return billy.ReadCapability
}

func NewFile(f fs.File) (*file, error) {
	if f == nil {
		return nil, fmt.Errorf("cannot create billy file from nil")
	}

	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("cannot stat file: %w", err)
	}

	return &file{f: f, name: fi.Name()}, nil
}

type file struct {
	name     string
	isClosed bool
	f        fs.File
}

func (f *file) Name() string {
	return f.name
}

func (f *file) Read(b []byte) (int, error) {
	if f == nil {
		return 0, fmt.Errorf("file is nil")
	}

	return f.f.Read(b)
}

func (f *file) ReadAt(b []byte, off int64) (int, error) {
	if f == nil {
		return 0, fmt.Errorf("file is nil")
	}

	if std, ok := f.f.(*os.File); ok {
		return std.ReadAt(b, off)
	}

	return 0, fmt.Errorf("readat not supported")
}

func (f *file) Seek(offset int64, whence int) (int64, error) {
	if f == nil {
		return 0, fmt.Errorf("file is nil")
	}
	if std, ok := f.f.(*os.File); ok {
		return std.Seek(offset, whence)
	}

	return 0, fmt.Errorf("seek not supported")
}

func (_ *file) Write(p []byte) (int, error) {
	return 0, fmt.Errorf("file write not supported: %w", billy.ErrReadOnly)
}

func (_ *file) Truncate(size int64) error {
	return fmt.Errorf("truncate not supported: %w", billy.ErrReadOnly)
}

func (f *file) Close() error {
	if f == nil {
		return fmt.Errorf("file is nil")
	}

	if f.isClosed {
		return os.ErrClosed
	}

	f.isClosed = true
	f.f.Close()
	return nil
}

func (f *file) Stat() (os.FileInfo, error) {
	if f == nil {
		return nil, fmt.Errorf("file is nil")
	}

	fi, err := f.f.Stat()
	if err != nil {
		return nil, err
	}

	return fi, nil
}

// Lock is a no-op in embedfs.
func (_ *file) Lock() error {
	return nil
}

// Unlock is a no-op in embedfs.
func (_ *file) Unlock() error {
	return nil
}
