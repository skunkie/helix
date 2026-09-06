// SPDX-FileCopyrightText: 2020 Ethel Morgan
// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package httpu

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSerializeRequest(t *testing.T) {
	tests := []struct {
		req  *http.Request
		want string
	}{
		{
			req: &http.Request{
				Method: "M-SEARCH",
				Host:   "239.255.255.250:1900",
				URL:    &url.URL{Opaque: "*"},
				Header: http.Header{
					"Man":        {`"ssdp:discover"`},
					"Mx":         {"2"},
					"St":         {"ssdp:all"},
					"User-Agent": {"Linux/3.x UPnP/1.0 Helix/1.0"},
				},
			},
			want: `M-SEARCH * HTTP/1.1
HOST: 239.255.255.250:1900
MAN: "ssdp:discover"
MX: 2
ST: ssdp:all
USER-AGENT: Linux/3.x UPnP/1.0 Helix/1.0
`,
		},
		{
			req: &http.Request{
				Method: "NOTIFY",
				Host:   "239.255.255.250:1900",
				URL:    &url.URL{Opaque: "*"},
				Header: http.Header{
					"location":        {"http://192.168.1.1:8080/upnp/"},
					"nt":              {"upnp:rootdevice"},
					"nts":             {"ssdp:alive"},
					"server":          {"Linux/3.x UPnP/1.0 Helix/1.0"},
					"usn":             {"uuid:1234::upnp:rootdevice"},
					"bootid.upnp.org": {"100"},
				},
			},
			want: `NOTIFY * HTTP/1.1
HOST: 239.255.255.250:1900
BOOTID.UPNP.ORG: 100
LOCATION: http://192.168.1.1:8080/upnp/
NT: upnp:rootdevice
NTS: ssdp:alive
SERVER: Linux/3.x UPnP/1.0 Helix/1.0
USN: uuid:1234::upnp:rootdevice
`,
		},
	}

	for i, tt := range tests {
		var want []byte
		for _, line := range strings.Split(tt.want, "\n") {
			want = append(want, []byte(line)...)
			want = append(want, []byte("\r\n")...)
		}

		got := SerializeRequest(tt.req)

		if !reflect.DeepEqual(got, want) {
			t.Errorf("[%d]: want:\n\n%s\n\ngot:\n\n%s", i, want, got)
		}
	}
}

func TestSendAndDoLoopback(t *testing.T) {
	serverConn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("net.ListenUDP failed: %v", err)
	}
	defer func() { _ = serverConn.Close() }()

	serverAddr := serverConn.LocalAddr().String()

	// Server goroutine: echo HTTP response
	go func() {
		buf := make([]byte, 2048)
		for {
			n, clientAddr, err := serverConn.ReadFrom(buf)
			if err != nil {
				return
			}
			if n > 0 {
				rsp := "HTTP/1.1 200 OK\r\nST: ssdp:all\r\nUSN: uuid:test\r\n\r\n"
				_, _ = serverConn.WriteTo([]byte(rsp), clientAddr)
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "M-SEARCH", "*", nil)
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}
	req.Host = serverAddr
	req.Header.Set("MAN", `"ssdp:discover"`)
	req.Header.Set("ST", "ssdp:all")

	// Test Send
	if err := Send(req, 1, nil); err != nil {
		t.Errorf("Send failed: %v", err)
	}

	// Test Do
	rsps, errs, err := Do(req, 1, nil)
	if err != nil {
		t.Errorf("Do returned connection error: %v", err)
	}
	if len(errs) > 0 {
		t.Errorf("Do returned response parse errors: %v", errs)
	}
	if len(rsps) == 0 {
		t.Errorf("Do returned 0 responses, expected at least 1")
	} else if st := rsps[0].Header.Get("ST"); st != "ssdp:all" {
		t.Errorf("Response ST = %q, want ssdp:all", st)
	}
}

func TestDoStopsWhenContextIsCanceled(t *testing.T) {
	serverConn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("net.ListenUDP failed: %v", err)
	}
	defer func() { _ = serverConn.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, "M-SEARCH", "*", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = serverConn.LocalAddr().String()
	time.AfterFunc(20*time.Millisecond, cancel)

	started := time.Now()
	_, _, err = Do(req, 1, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Do error = %v, want context.Canceled", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("Do took %v to observe cancellation", elapsed)
	}
}
