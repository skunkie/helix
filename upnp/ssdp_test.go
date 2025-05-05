// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

package upnp

import (
	"context"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/ethulhu/helix/upnp/httpu"
)

func TestHandleDiscover(t *testing.T) {
	tests := []struct {
		req    *http.Request
		device *Device
		url    string
		want   []httpu.Response
	}{
		{
			req: &http.Request{
				Method: "M-SEARCH",
				Host:   "239.255.255.250:1900",
				URL:    &url.URL{Opaque: "*"},
				Header: http.Header{
					"Man": {`"ssdp:discover"`},
					"Mx":  {"2"},
					"St":  {"ssdp:all"},
				},
			},
			device: &Device{
				DeviceType: DeviceType("device-type"),
				UDN:        "device-id",
				bootID:     123,
			},
			url: "http://1.2.3.4:8000/",
			want: []httpu.Response{
				{
					"CACHE-CONTROL":   ssdpCacheControl,
					"EXT":             "",
					"LOCATION":        "http://1.2.3.4:8000/",
					"SERVER":          " ",
					"ST":              "device-id",
					"USN":             "device-id",
					"BOOTID.UPNP.ORG": "123",
				},
				{
					"CACHE-CONTROL":   ssdpCacheControl,
					"EXT":             "",
					"LOCATION":        "http://1.2.3.4:8000/",
					"SERVER":          " ",
					"ST":              "device-type",
					"USN":             "device-id::device-type",
					"BOOTID.UPNP.ORG": "123",
				},
				{
					"CACHE-CONTROL":   ssdpCacheControl,
					"EXT":             "",
					"LOCATION":        "http://1.2.3.4:8000/",
					"SERVER":          " ",
					"ST":              "upnp:rootdevice",
					"USN":             "device-id::upnp:rootdevice",
					"BOOTID.UPNP.ORG": "123",
				},
			},
		},
		{
			req: &http.Request{
				Method: "M-SEARCH",
				Host:   "239.255.255.250:1900",
				URL:    &url.URL{Opaque: "*"},
				Header: http.Header{
					"Man": {`"ssdp:discover"`},
					"Mx":  {"2"},
					"St":  {"device-type"},
				},
			},
			device: &Device{
				DeviceType: DeviceType("device-type"),
				UDN:        "device-id",
				bootID:     123,
			},
			url: "http://1.2.3.4:8000/",
			want: []httpu.Response{
				{
					"CACHE-CONTROL":   ssdpCacheControl,
					"EXT":             "",
					"LOCATION":        "http://1.2.3.4:8000/",
					"SERVER":          " ",
					"ST":              "device-id",
					"USN":             "device-id",
					"BOOTID.UPNP.ORG": "123",
				},
				{
					"CACHE-CONTROL":   ssdpCacheControl,
					"EXT":             "",
					"LOCATION":        "http://1.2.3.4:8000/",
					"SERVER":          " ",
					"ST":              "device-type",
					"USN":             "device-id::device-type",
					"BOOTID.UPNP.ORG": "123",
				},
				{
					"CACHE-CONTROL":   ssdpCacheControl,
					"EXT":             "",
					"LOCATION":        "http://1.2.3.4:8000/",
					"SERVER":          " ",
					"ST":              "upnp:rootdevice",
					"USN":             "device-id::upnp:rootdevice",
					"BOOTID.UPNP.ORG": "123",
				},
			},
		},
		{
			req: &http.Request{
				Method: "M-SEARCH",
				Host:   "239.255.255.250:1900",
				URL:    &url.URL{Opaque: "*"},
				Header: http.Header{
					"Man": {`"ssdp:discover"`},
					"Mx":  {"2"},
					"St":  {"service-urn"},
				},
			},
			device: &Device{
				DeviceType: DeviceType("device-type"),
				UDN:        "device-id",
				bootID:     123,
				serviceByURN: map[URN]service{
					"service-urn": service{},
				},
			},
			url: "http://1.2.3.4:8000/",
			want: []httpu.Response{
				{
					"CACHE-CONTROL":   ssdpCacheControl,
					"EXT":             "",
					"LOCATION":        "http://1.2.3.4:8000/",
					"SERVER":          " ",
					"ST":              "device-id",
					"USN":             "device-id",
					"BOOTID.UPNP.ORG": "123",
				},
				{
					"CACHE-CONTROL":   ssdpCacheControl,
					"EXT":             "",
					"LOCATION":        "http://1.2.3.4:8000/",
					"SERVER":          " ",
					"ST":              "service-urn",
					"USN":             "device-id::service-urn",
					"BOOTID.UPNP.ORG": "123",
				},
				{
					"CACHE-CONTROL":   ssdpCacheControl,
					"EXT":             "",
					"LOCATION":        "http://1.2.3.4:8000/",
					"SERVER":          " ",
					"ST":              "device-type",
					"USN":             "device-id::device-type",
					"BOOTID.UPNP.ORG": "123",
				},
				{
					"CACHE-CONTROL":   ssdpCacheControl,
					"EXT":             "",
					"LOCATION":        "http://1.2.3.4:8000/",
					"SERVER":          " ",
					"ST":              "upnp:rootdevice",
					"USN":             "device-id::upnp:rootdevice",
					"BOOTID.UPNP.ORG": "123",
				},
			},
		},
		{
			req: &http.Request{
				Method: "M-SEARCH",
				Host:   "239.255.255.250:1900",
				URL:    &url.URL{Opaque: "*"},
				Header: http.Header{
					"Man": {`"ssdp:discover"`},
					"Mx":  {"2"},
					"St":  {"twaddle"},
				},
			},
			device: &Device{
				DeviceType: DeviceType("meowpurr"),
				UDN:        "foobar",
				serviceByURN: map[URN]service{
					"tweedle": service{},
				},
			},
			url:  "http://1.2.3.4:8000/",
			want: nil,
		},
	}

	for i, tt := range tests {
		got := handleDiscover(tt.req, tt.device, tt.url)

		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("[%d]: got:\n\n%v\n\nwant:\n\n%v", i, got, tt.want)
		}
	}
}

func TestNotifyUpdateRequest(t *testing.T) {
	device := &Device{
		UDN: "device-id",
	}
	device.SetBootID(123)
	url := "http://1.2.3.4:8000/"

	req := notifyUpdateRequest(context.Background(), device, url)

	if req.Method != notifyMethod {
		t.Errorf("got method %q, want %q", req.Method, notifyMethod)
	}
	if req.Host != ssdpBroadcastAddr.String() {
		t.Errorf("got host %q, want %q", req.Host, ssdpBroadcastAddr.String())
	}
	if req.URL.Opaque != "*" {
		t.Errorf("got URL %q, want %q", req.URL.Opaque, "*")
	}

	headers := http.Header{
		"Location":            {url},
		"Nt":                  {string(RootDevice)},
		"Nts":                 {notifyUpdate},
		"Usn":                 {device.UDN},
		"BootID.upnp.org":     {"123"},
		"NextBootID.upnp.org": {"124"},
	}
	if !reflect.DeepEqual(req.Header, headers) {
		t.Errorf("got headers:\n%#v\nwant:\n%#v", req.Header, headers)
	}
}

func TestNotifyByeByeRequests(t *testing.T) {
	device := &Device{
		DeviceType: "device-type",
		UDN:        "device-id",
		serviceByURN: map[URN]service{
			"service-urn": {},
		},
	}
	device.SetBootID(123)

	reqs := notifyByeByeRequests(context.Background(), device)

	want := []*http.Request{}
	for _, urn := range []URN{"service-urn", "device-type", "upnp:rootdevice"} {
		header := http.Header{
			"Nt":              {string(urn)},
			"Nts":             {notifyByeBye},
			"Usn":             {"device-id::" + string(urn)},
			"BootID.upnp.org": {"123"},
		}
		req, _ := http.NewRequest(notifyMethod, "", nil)
		req.URL = discoverURL
		req.Host = ssdpBroadcastAddr.String()
		req.Header = header
		want = append(want, req)
	}

	if len(reqs) != len(want) {
		t.Fatalf("got %d requests, want %d", len(reqs), len(want))
	}

	for i := range reqs {
		if !reflect.DeepEqual(reqs[i].Header, want[i].Header) {
			t.Errorf("Request %d: got headers:\n%#v\nwant:\n%#v", i, reqs[i].Header, want[i].Header)
		}
	}
}
