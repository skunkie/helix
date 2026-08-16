// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package upnpav

import (
	"encoding/xml"
	"net/url"
	"reflect"
	"testing"
)

func TestParseURL(t *testing.T) {
	tests := []struct {
		raw     string
		want    *URL
		wantErr bool
	}{
		{
			raw: "http://example.com/icon.png",
			want: &URL{url.URL{
				Scheme: "http",
				Host:   "example.com",
				Path:   "/icon.png",
			}},
		},
		{
			raw: "https://example.com:8080/path?foo=bar#frag",
			want: &URL{url.URL{
				Scheme:   "https",
				Host:     "example.com:8080",
				Path:     "/path",
				RawQuery: "foo=bar",
				Fragment: "frag",
			}},
		},
		{
			raw:     "://invalid-url",
			wantErr: true,
		},
	}

	for i, tt := range tests {
		got, err := ParseURL(tt.raw)
		if (err != nil) != tt.wantErr {
			t.Errorf("[%d]: ParseURL(%q) err = %v, wantErr = %v", i, tt.raw, err, tt.wantErr)
		}
		if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
			t.Errorf("[%d]: ParseURL(%q) = %v, want %v", i, tt.raw, got, tt.want)
		}
	}
}

func TestURLMarshalXML(t *testing.T) {
	u, _ := ParseURL("http://example.com/album.png")
	doc := urlTestDocument{
		Icon: u,
	}

	bytes, err := xml.Marshal(doc)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	want := `<test><icon>http://example.com/album.png</icon></test>`
	if string(bytes) != want {
		t.Errorf("got %s, want %s", string(bytes), want)
	}
}

func TestURLUnmarshalXML(t *testing.T) {
	raw := `<test><icon>http://example.com/album.png</icon></test>`
	var got urlTestDocument
	if err := xml.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if got.Icon == nil || got.Icon.String() != "http://example.com/album.png" {
		t.Errorf("got %v, want http://example.com/album.png", got.Icon)
	}
}

type urlTestDocument struct {
	XMLName xml.Name `xml:"test"`
	Icon    *URL     `xml:"icon,omitempty"`
}
