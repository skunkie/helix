// SPDX-FileCopyrightText: 2020 Ethel Morgan
// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package upnpav

import (
	"reflect"
	"testing"
)

func TestParseDIDLLite(t *testing.T) {
	tests := []struct {
		raw     string
		want    *DIDLLite
		wantErr error
	}{
		{
			raw: `<?xml version="1.0" encoding="UTF-8"?><DIDL-Lite xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:upnp="urn:schemas-upnp-org:metadata-1-0/upnp/" xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/" xmlns:dlna="urn:schemas-dlna-org:metadata-1-0/"><item restricted="1" searchable="0"><res protocolInfo="http-get:*:audio/mpeg:*">http://192.168.16.4:8200/MediaItems/36.mp3</res></item></DIDL-Lite>`,
			want: &DIDLLite{
				Items: []Item{{
					Restricted: true,
					Searchable: false,
					Resources: []Resource{{
						URI: "http://192.168.16.4:8200/MediaItems/36.mp3",
						ProtocolInfo: &ProtocolInfo{
							Protocol:       ProtocolHTTP,
							Network:        "*",
							ContentFormat:  "audio/mpeg",
							AdditionalInfo: "*",
						},
					}},
				}},
			},
		},

		{
			raw: `
<DIDL-Lite xmlns:dc="http://purl.org/dc/elements/1.1/"
xmlns:upnp="urn:schemas-upnp-org:metadata-1-0/upnp/"
xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/"
xmlns:dlna="urn:schemas-dlna-org:metadata-1-0/">
  <container id="64" parentID="0" restricted="1" searchable="1" childCount="4">
    <dc:title>Browse Folders</dc:title>
    <upnp:class>object.container.storageFolder</upnp:class>
    <upnp:storageUsed>-1</upnp:storageUsed>
  </container>
  <container id="1" parentID="0" restricted="0" searchable="0" childCount="7">
    <dc:title>Music</dc:title>
    <upnp:class>object.container.storageFolder</upnp:class>
    <upnp:storageUsed>-1</upnp:storageUsed>
  </container>
  <item id="72" parentID="4" restricted="0" searchable="0">
    <res protocolInfo="http-get:*:audio/mpeg:*" colorDepth="3">http://mew/purr.mp3</res>
    <res protocolInfo="http-get:*:video/mp4:*" resolution="480x360">http://mew/purr.mp4</res>
  </item>
</DIDL-Lite>
`,
			want: &DIDLLite{
				Containers: []Container{
					{
						ID:               ObjectID("64"),
						Parent:           ObjectID("0"),
						Restricted:       true,
						Searchable:       true,
						Title:            "Browse Folders",
						Class:            StorageFolder,
						ChildCount:       4,
						StorageUsedBytes: -1,
					},
					{
						ID:               ObjectID("1"),
						Parent:           ObjectID("0"),
						Restricted:       false,
						Searchable:       false,
						Title:            "Music",
						Class:            StorageFolder,
						ChildCount:       7,
						StorageUsedBytes: -1,
					},
				},
				Items: []Item{
					{
						ID:         ObjectID("72"),
						Parent:     ObjectID("4"),
						Restricted: false,
						Searchable: false,
						Resources: []Resource{
							{
								URI: "http://mew/purr.mp3",
								ProtocolInfo: &ProtocolInfo{
									Protocol:       ProtocolHTTP,
									Network:        "*",
									ContentFormat:  "audio/mpeg",
									AdditionalInfo: "*",
								},
								ColorDepth: 3,
							},
							{
								URI: "http://mew/purr.mp4",
								ProtocolInfo: &ProtocolInfo{
									Protocol:       ProtocolHTTP,
									Network:        "*",
									ContentFormat:  "video/mp4",
									AdditionalInfo: "*",
								},
								Resolution: &Resolution{
									Width:  480,
									Height: 360,
								},
							},
						},
					},
				},
			},
		},
	}

	for i, tt := range tests {
		got, gotErr := ParseDIDLLite(tt.raw)
		if !reflect.DeepEqual(tt.wantErr, gotErr) {
			t.Errorf("[%d]: expected error %v, got %v", i, tt.wantErr, gotErr)
		}
		if !reflect.DeepEqual(tt.want, got) {
			t.Errorf("[%d]: got:\n\n%v\n\nwant:\n\n%v\n\n", i, got, tt.want)
		}
	}
}

func TestMarshalDIDLLite(t *testing.T) {
	tests := []struct {
		didllite *DIDLLite
		want     string
	}{
		{
			didllite: &DIDLLite{},
			want:     `<DIDL-Lite xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:upnp="urn:schemas-upnp-org:metadata-1-0/upnp/" xmlns:dlna="urn:schemas-dlna-org:metadata-1-0/"></DIDL-Lite>`,
		},
		{
			didllite: &DIDLLite{
				Containers: []Container{
					{
						ID:               ObjectID("64"),
						Parent:           ObjectID("0"),
						Restricted:       false,
						Searchable:       true,
						Title:            "Browse Folders",
						Class:            StorageFolder,
						ChildCount:       4,
						StorageUsedBytes: -1,
					},
					{
						ID:               ObjectID("1"),
						Parent:           ObjectID("0"),
						Restricted:       true,
						Searchable:       false,
						Title:            "Music",
						Class:            StorageFolder,
						ChildCount:       7,
						StorageUsedBytes: -1,
					},
				},
			},
			want: `<DIDL-Lite xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:upnp="urn:schemas-upnp-org:metadata-1-0/upnp/" xmlns:dlna="urn:schemas-dlna-org:metadata-1-0/">
  <container id="64" parentID="0" restricted="0" searchable="1" childCount="4">
    <dc:title>Browse Folders</dc:title>
    <upnp:class>object.container.storageFolder</upnp:class>
    <upnp:storageUsed>-1</upnp:storageUsed>
  </container>
  <container id="1" parentID="0" restricted="1" searchable="0" childCount="7">
    <dc:title>Music</dc:title>
    <upnp:class>object.container.storageFolder</upnp:class>
    <upnp:storageUsed>-1</upnp:storageUsed>
  </container>
</DIDL-Lite>`,
		},
		{
			didllite: &DIDLLite{
				Items: []Item{
					{
						ID:         ObjectID("69"),
						Parent:     ObjectID("12"),
						Restricted: false,
						Searchable: true,
						Title:      "hello",
						Resources: []Resource{
							{
								URI: "http://mew/purr.mp3",
								ProtocolInfo: &ProtocolInfo{
									Protocol:      ProtocolHTTP,
									ContentFormat: "audio/mpeg",
								},
								BitsPerSecond: 128 * 1024,
							},
							{
								URI: "http://mew/purr.mp4",
								ProtocolInfo: &ProtocolInfo{
									Protocol:      ProtocolHTTP,
									ContentFormat: "video/mp4",
								},
								Resolution: &Resolution{
									Width:  480,
									Height: 360,
								},
							},
						},
					},
				},
			},
			want: `<DIDL-Lite xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:upnp="urn:schemas-upnp-org:metadata-1-0/upnp/" xmlns:dlna="urn:schemas-dlna-org:metadata-1-0/">
  <item id="69" parentID="12" restricted="0" searchable="1">
    <dc:title>hello</dc:title>
    <res protocolInfo="http-get:*:audio/mpeg:*" bitrate="131072">http://mew/purr.mp3</res>
    <res protocolInfo="http-get:*:video/mp4:*" resolution="480x360">http://mew/purr.mp4</res>
  </item>
</DIDL-Lite>`,
		},
	}

	for i, tt := range tests {
		got := tt.didllite.String()
		if tt.want != got {
			t.Errorf("[%d]: got:\n\n%+v\n\nwant:\n\n%+v", i, got, tt.want)
		}
	}
}

func TestDIDLLitePaginate(t *testing.T) {
	sample := &DIDLLite{
		Containers: []Container{
			{ID: "c1", Title: "Container 1"},
			{ID: "c2", Title: "Container 2"},
		},
		Items: []Item{
			{ID: "i1", Title: "Item 1"},
			{ID: "i2", Title: "Item 2"},
			{ID: "i3", Title: "Item 3"},
		},
	}

	tests := []struct {
		name           string
		startingIndex  uint
		requestedCount uint
		wantContainers []string
		wantItems      []string
	}{
		{
			name:           "all elements (count 0)",
			startingIndex:  0,
			requestedCount: 0,
			wantContainers: []string{"c1", "c2"},
			wantItems:      []string{"i1", "i2", "i3"},
		},
		{
			name:           "first page across containers only",
			startingIndex:  0,
			requestedCount: 1,
			wantContainers: []string{"c1"},
			wantItems:      nil,
		},
		{
			name:           "span containers and items",
			startingIndex:  1,
			requestedCount: 2,
			wantContainers: []string{"c2"},
			wantItems:      []string{"i1"},
		},
		{
			name:           "items only start from item 0",
			startingIndex:  2,
			requestedCount: 2,
			wantContainers: nil,
			wantItems:      []string{"i1", "i2"},
		},
		{
			name:           "items with offset",
			startingIndex:  3,
			requestedCount: 2,
			wantContainers: nil,
			wantItems:      []string{"i2", "i3"},
		},
		{
			name:           "items with count exceeding remainder",
			startingIndex:  4,
			requestedCount: 5,
			wantContainers: nil,
			wantItems:      []string{"i3"},
		},
		{
			name:           "maximum count does not overflow",
			startingIndex:  1,
			requestedCount: ^uint(0),
			wantContainers: []string{"c2"},
			wantItems:      []string{"i1", "i2", "i3"},
		},
		{
			name:           "starting index past total elements",
			startingIndex:  10,
			requestedCount: 5,
			wantContainers: nil,
			wantItems:      nil,
		},
		{
			name:           "offset from container to end (count 0)",
			startingIndex:  1,
			requestedCount: 0,
			wantContainers: []string{"c2"},
			wantItems:      []string{"i1", "i2", "i3"},
		},
		{
			name:           "offset from item to end (count 0)",
			startingIndex:  3,
			requestedCount: 0,
			wantContainers: nil,
			wantItems:      []string{"i2", "i3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sample.Paginate(tt.startingIndex, tt.requestedCount)
			var gotContainers []string
			for _, c := range got.Containers {
				gotContainers = append(gotContainers, string(c.ID))
			}
			var gotItems []string
			for _, item := range got.Items {
				gotItems = append(gotItems, string(item.ID))
			}

			if !reflect.DeepEqual(gotContainers, tt.wantContainers) {
				t.Errorf("Containers = %v, want %v", gotContainers, tt.wantContainers)
			}
			if !reflect.DeepEqual(gotItems, tt.wantItems) {
				t.Errorf("Items = %v, want %v", gotItems, tt.wantItems)
			}
		})
	}

	// Test nil receiver
	var nilDIDL *DIDLLite
	if got := nilDIDL.Paginate(0, 5); got != nil {
		t.Errorf("nil.Paginate() = %v, want nil", got)
	}
}

func TestDIDLLiteHelpers(t *testing.T) {
	cOnly := DIDLLite{Containers: []Container{{ID: "c1"}}}
	iOnly := DIDLLite{Items: []Item{{ID: "i1"}}}
	both := DIDLLite{Containers: []Container{{ID: "c1"}}, Items: []Item{{ID: "i1"}}}
	empty := DIDLLite{}

	if !cOnly.IsSingleContainer() || iOnly.IsSingleContainer() || both.IsSingleContainer() || empty.IsSingleContainer() {
		t.Errorf("IsSingleContainer failed")
	}

	if !iOnly.IsSingleItem() || cOnly.IsSingleItem() || both.IsSingleItem() || empty.IsSingleItem() {
		t.Errorf("IsSingleItem failed")
	}
}

func TestEncodedDIDLLite(t *testing.T) {
	original := EncodedDIDLLite{
		DIDLLite: DIDLLite{
			Items: []Item{
				{
					ID:    "item-1",
					Title: "Sample Track",
					Class: "object.item.audioItem.musicTrack",
				},
			},
		},
	}

	text, err := original.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText error: %v", err)
	}

	var decoded EncodedDIDLLite
	if err := decoded.UnmarshalText(text); err != nil {
		t.Fatalf("UnmarshalText error: %v", err)
	}

	if len(decoded.Items) != 1 || decoded.Items[0].Title != "Sample Track" {
		t.Errorf("decoded = %v, want original items", decoded)
	}

	var badDecoded EncodedDIDLLite
	if err := badDecoded.UnmarshalText([]byte("invalid XML <<>>")); err == nil {
		t.Errorf("expected unmarshal error for invalid XML, got nil")
	}
}

func TestDIDLForURI(t *testing.T) {
	didl, err := DIDLForURI("http://example.com/movie.mp4")
	if err != nil {
		t.Fatalf("DIDLForURI failed: %v", err)
	}
	if len(didl.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(didl.Items))
	}
	item := didl.Items[0]
	if item.Class != "object.item.videoItem" {
		t.Errorf("item.Class = %v, want videoItem", item.Class)
	}
	if !item.HasURI("http://example.com/movie.mp4") {
		t.Errorf("HasURI(movie.mp4) = false, want true")
	}
	if item.HasURI("http://example.com/other.mp4") {
		t.Errorf("HasURI(other.mp4) = true, want false")
	}

	pInfo := item.Resources[0].ProtocolInfo
	matchedURI, ok := item.URIForProtocolInfos([]ProtocolInfo{*pInfo})
	if !ok || matchedURI != "http://example.com/movie.mp4" {
		t.Errorf("URIForProtocolInfos = %v, %v, want http://example.com/movie.mp4, true", matchedURI, ok)
	}

	_, ok = item.URIForProtocolInfos([]ProtocolInfo{{Protocol: "rtsp", ContentFormat: "video/mp4"}})
	if ok {
		t.Errorf("URIForProtocolInfos with unmatched protocol expected false, got true")
	}

	item.Resources = append([]Resource{{URI: "missing-protocol-info"}}, item.Resources...)
	matchedURI, ok = item.URIForProtocolInfos([]ProtocolInfo{*pInfo})
	if !ok || matchedURI != "http://example.com/movie.mp4" {
		t.Errorf("URIForProtocolInfos with nil ProtocolInfo = %v, %v", matchedURI, ok)
	}
}
