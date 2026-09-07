// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

package fileserver

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/ethulhu/helix/media"
	"github.com/ethulhu/helix/upnpav/contentdirectory"
	"github.com/ethulhu/helix/upnpav/contentdirectory/search"
	"github.com/ethulhu/helix/xmltypes"
)

type mockMetadataCache struct{}

type nilMetadataCache struct{ mockMetadataCache }

func (*nilMetadataCache) MetadataForPaths(paths []string) []*media.Metadata {
	return make([]*media.Metadata, len(paths))
}

func (m *mockMetadataCache) MetadataForPath(p string) (*media.Metadata, error) {
	fileName := filepath.Base(p)
	var mimeType string
	switch filepath.Ext(fileName) {
	case ".mp4":
		mimeType = "video/mp4"
	case ".mkv":
		mimeType = "video/x-matroska"
	case ".mp3":
		mimeType = "audio/mpeg"
	default:
		return nil, fmt.Errorf("unsupported for mock: %v", fileName)
	}
	return &media.Metadata{
		Title:     fileName,
		MIMEType:  mimeType,
		Duration:  1 * time.Second,
		SizeBytes: 1234,
	}, nil
}

func TestSearchSortsBeforePagination(t *testing.T) {
	tmpdir := t.TempDir()
	for _, file := range []string{"charlie.mp3", "alpha.mp3", "bravo.mp3"} {
		if err := os.WriteFile(filepath.Join(tmpdir, file), []byte("dummy content"), 0600); err != nil {
			t.Fatalf("creating dummy file: %v", err)
		}
	}

	baseURL, err := url.Parse("http://localhost/")
	if err != nil {
		t.Fatalf("parsing base URL: %v", err)
	}
	cd := &ContentDirectory{basePath: tmpdir, baseURL: baseURL, metadataCache: &mockMetadataCache{}}
	criteria, err := search.Parse("*")
	if err != nil {
		t.Fatalf("parsing criteria: %v", err)
	}

	result, total, err := cd.Search(context.Background(), "0", criteria, 1, 1, xmltypes.CommaSeparatedStrings{"-dc:title"})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}
	if len(result.Items) != 1 || result.Items[0].Title != "bravo.mp3" {
		t.Fatalf("items = %+v, want the second descending title", result.Items)
	}

	_, _, err = cd.Search(context.Background(), "0", criteria, 0, 0, xmltypes.CommaSeparatedStrings{"+dc:date"})
	if !errors.Is(err, contentdirectory.ErrInvalidSortCriteria) {
		t.Fatalf("unsupported sort error = %v, want %v", err, contentdirectory.ErrInvalidSortCriteria)
	}
}

func TestBrowseChildrenSortsSupportedObjects(t *testing.T) {
	tmpdir := t.TempDir()
	for _, directory := range []string{"charlie", "alpha"} {
		if err := os.Mkdir(filepath.Join(tmpdir, directory), 0755); err != nil {
			t.Fatalf("creating directory: %v", err)
		}
	}
	for _, file := range []string{"delta.mp3", "bravo.mp3"} {
		if err := os.WriteFile(filepath.Join(tmpdir, file), []byte("dummy content"), 0600); err != nil {
			t.Fatalf("creating dummy file: %v", err)
		}
	}

	baseURL, err := url.Parse("http://localhost/")
	if err != nil {
		t.Fatalf("parsing base URL: %v", err)
	}
	cd := &ContentDirectory{basePath: tmpdir, baseURL: baseURL, metadataCache: &mockMetadataCache{}}

	result, total, err := cd.BrowseChildren(context.Background(), contentdirectory.Root, 0, 0, xmltypes.CommaSeparatedStrings{"+dc:title"})
	if err != nil {
		t.Fatalf("BrowseChildren failed: %v", err)
	}
	if total != 4 {
		t.Fatalf("total = %d, want 4", total)
	}
	if got := []string{result.Containers[0].Title, result.Containers[1].Title}; !reflect.DeepEqual(got, []string{"alpha", "charlie"}) {
		t.Fatalf("container titles = %v", got)
	}
	if got := []string{result.Items[0].Title, result.Items[1].Title}; !reflect.DeepEqual(got, []string{"bravo.mp3", "delta.mp3"}) {
		t.Fatalf("item titles = %v", got)
	}

	_, _, err = cd.BrowseChildren(context.Background(), contentdirectory.Root, 0, 0, xmltypes.CommaSeparatedStrings{"+dc:date"})
	if !errors.Is(err, contentdirectory.ErrInvalidSortCriteria) {
		t.Fatalf("unsupported sort error = %v, want %v", err, contentdirectory.ErrInvalidSortCriteria)
	}
}

func TestItemsForPathsSkipsMissingMetadata(t *testing.T) {
	baseURL, err := url.Parse("http://localhost/")
	if err != nil {
		t.Fatal(err)
	}
	cd := &ContentDirectory{basePath: t.TempDir(), baseURL: baseURL, metadataCache: &nilMetadataCache{}}

	items, err := cd.itemsForPaths(filepath.Join(cd.basePath, "missing.mp3"))
	if err != nil {
		t.Fatalf("itemsForPaths failed: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("itemsForPaths returned %+v, want no items", items)
	}
}

func (m *mockMetadataCache) MetadataForPaths(paths []string) []*media.Metadata {
	var mds []*media.Metadata
	for _, p := range paths {
		md, _ := m.MetadataForPath(p)
		mds = append(mds, md)
	}
	return mds
}

func (m *mockMetadataCache) Warm(p string) {}

func TestSearch(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "helix-fileserver-test")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpdir) }()

	files := []string{
		"video1.mp4",
		"audio1.mp3",
		"video2.mkv",
		"not-media.txt",
	}
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(tmpdir, file), []byte("dummy content"), 0600); err != nil {
			t.Fatalf("creating dummy file: %v", err)
		}
	}

	baseURL, err := url.Parse("http://localhost/")
	if err != nil {
		t.Fatalf("parsing base URL: %v", err)
	}
	cd := &ContentDirectory{
		basePath:      tmpdir,
		baseURL:       baseURL,
		metadataCache: &mockMetadataCache{},
	}

	tests := []struct {
		name       string
		criteria   string
		wantTitles []string
	}{
		{
			name:       "Search for videos",
			criteria:   `upnp:class derivedfrom "object.item.videoItem"`,
			wantTitles: []string{"video1.mp4", "video2.mkv"},
		},
		{
			name:       "Search for audio",
			criteria:   `upnp:class derivedfrom "object.item.audioItem"`,
			wantTitles: []string{"audio1.mp3"},
		},
		{
			name:       "Search by title",
			criteria:   `dc:title = "video1.mp4"`,
			wantTitles: []string{"video1.mp4"},
		},
		{
			name:       "No results",
			criteria:   `dc:title = "nonexistent"`,
			wantTitles: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crit, err := search.Parse(tt.criteria)
			if err != nil {
				t.Fatalf("parsing criteria: %v", err)
			}
			result, _, err := cd.Search(context.Background(), "0", crit, 0, 0, nil)
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}

			if result == nil {
				t.Fatal("result is nil")
			}

			var gotTitles []string
			for _, item := range result.Items {
				gotTitles = append(gotTitles, item.Title)
			}

			sort.Strings(gotTitles)
			if tt.wantTitles != nil {
				sort.Strings(tt.wantTitles)
			}

			if !reflect.DeepEqual(gotTitles, tt.wantTitles) {
				t.Errorf("got titles %v, want %v", gotTitles, tt.wantTitles)
			}
		})
	}
}
