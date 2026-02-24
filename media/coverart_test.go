// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

package media

import (
	"os"
	"path"
	"reflect"
	"sort"
	"testing"
)

func TestCoverArtForPath(t *testing.T) {
	tests := []struct {
		fs    fakeFS
		paths []string
		want  [][]string
	}{
		{
			fs: fakeFS{
				"/music":            true,
				"/music/folder.jpg": false,
			},
			paths: []string{"/music"},
			want: [][]string{
				{"/music/folder.jpg"},
			},
		},
		{
			fs: fakeFS{
				"/music": true,
			},
			paths: []string{"/music"},
			want:  [][]string{nil},
		},
		{
			fs: fakeFS{
				"/music":            true,
				"/music/folder.jpg": false,
			},
			paths: []string{"/music"},
			want: [][]string{
				{"/music/folder.jpg"},
			},
		},
		{
			fs: fakeFS{
				"/music":            true,
				"/music/folder.jpg": false,
				"/music/foo.mp3":    false,
			},
			paths: []string{"/music/foo.mp3"},
			want: [][]string{
				{"/music/folder.jpg"},
			},
		},
		{
			fs: fakeFS{
				"/music":             true,
				"/music/folder.jpg":  false,
				"/music/foo.mp3":     false,
				"/music/foo.png":     false,
				"/music/foo.mp3.jpg": false,
			},
			paths: []string{"/music/foo.mp3"},
			want: [][]string{
				{"/music/foo.mp3.jpg", "/music/foo.png"},
			},
		},
		{
			fs: fakeFS{
				"/music":             true,
				"/music/folder.jpg":  false,
				"/music/foo.mp3":     false,
				"/music/foo.mp3.jpg": false,
				"/music/foo 2.mp3":   false,
				"/music/foo 2.png":   false,
			},
			paths: []string{"/music/foo.mp3"},
			want: [][]string{
				{"/music/foo.mp3.jpg"},
			},
		},
	}

	for i, tt := range tests {
		got := coverArtForPaths(tt.fs, tt.paths)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("[%d]: coverArtForPath(_, %q) == %q, want %q", i, tt.paths, got, tt.want)
		}
	}
}

// true == directory, false == file
type fakeFS map[string]bool

func (fs fakeFS) Stat(p string) (os.FileInfo, error) {
	p = path.Clean(p)
	if isDir, ok := fs[p]; ok {
		return fakeFileInfo{name: path.Base(p), isDir: isDir}, nil
	}
	return nil, os.ErrNotExist
}
func (fs fakeFS) List(p string) ([]os.DirEntry, error) {
	p = path.Clean(p)
	isDir, ok := fs[p]
	if !ok {
		return nil, os.ErrNotExist
	}
	if !isDir {
		return nil, os.ErrInvalid
	}

	var entries []os.DirEntry
	for f, isDir := range fs {
		if p == path.Dir(f) {
			entries = append(entries, fakeDirEntry{entryName: path.Base(f), entryIsDir: isDir})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
	return entries, nil
}

type fakeFileInfo struct {
	os.FileInfo
	name  string
	isDir bool
}

func (ffi fakeFileInfo) IsDir() bool {
	return ffi.isDir
}
func (ffi fakeFileInfo) Name() string {
	return ffi.name
}

type fakeDirEntry struct {
	entryName string
	entryIsDir bool
}

func (e fakeDirEntry) Name() string {
	return e.entryName
}

func (e fakeDirEntry) IsDir() bool {
	return e.entryIsDir
}

func (e fakeDirEntry) Type() os.FileMode {
	if e.entryIsDir {
		return os.ModeDir
	}
	return 0
}

func (e fakeDirEntry) Info() (os.FileInfo, error) {
	return fakeFileInfo{name: e.entryName, isDir: e.entryIsDir}, nil
}
