// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

package fileserver

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/ethulhu/helix/media"
	"github.com/ethulhu/helix/upnpav/contentdirectory/search"
)

type mockMetadataCache struct{}

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
		Title:    fileName,
		MIMEType: mimeType,
		Duration: 1 * time.Second,
		SizeBytes: 1234,
	}, nil
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
	defer os.RemoveAll(tmpdir)

	files := []string{
		"video1.mp4",
		"audio1.mp3",
		"video2.mkv",
		"not-media.txt",
	}
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(tmpdir, file), []byte("dummy content"), 0644); err != nil {
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
			result, err := cd.Search(context.Background(), "0", crit)
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
