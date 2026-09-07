// SPDX-FileCopyrightText: 2020 Ethel Morgan
// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package httpu

import (
	"bytes"
	"net"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSerializeResponse(t *testing.T) {
	tests := []struct {
		rsp  Response
		want string
	}{
		{
			rsp: Response{
				"LOCATION": "http://foo.com",
				"USN":      "uuid:6006::upnp:rootdevice",
				"EXT":      "",
			},
			want: `HTTP/1.1 200 OK
EXT:
LOCATION: http://foo.com
USN: uuid:6006::upnp:rootdevice
`,
		},
	}

	for i, tt := range tests {
		var want []byte
		for line := range strings.SplitSeq(tt.want, "\n") {
			want = append(want, []byte(line)...)
			want = append(want, []byte("\r\n")...)
		}

		got := tt.rsp.Bytes()
		if !reflect.DeepEqual(got, want) {
			t.Errorf("[%d]: want:\n\n%s\n\ngot:\n\n%s", i, want, got)
		}
	}
}

func TestServerRunningLifecycle(t *testing.T) {
	s := &Server{}

	if s.Running() {
		t.Error("Running() should be false before Serve()")
	}

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("net.ListenUDP failed: %v", err)
	}
	defer func() { _ = conn.Close() }()

	s.Handler = func(req *http.Request) []Response {
		return nil
	}

	done := make(chan struct{})
	go func() {
		_ = s.Serve(conn)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)

	if !s.Running() {
		t.Error("Running() should be true during Serve()")
	}

	if err := s.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("Serve() did not return after Close()")
	}

	if s.Running() {
		t.Error("Running() should be false after Close()")
	}
}

func TestServerHandlesRequest(t *testing.T) {
	handler := &mockHandler{}
	s := &Server{Handler: handler.Handle}

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("net.ListenUDP failed: %v", err)
	}

	clientConn, err := net.DialUDP("udp4", nil, conn.LocalAddr().(*net.UDPAddr))
	if err != nil {
		t.Fatalf("net.DialUDP failed: %v", err)
	}
	defer func() { _ = clientConn.Close() }()

	go func() {
		_ = s.Serve(conn)
	}()

	time.Sleep(50 * time.Millisecond)

	req := "NOTIFY * HTTP/1.1\r\nHOST: 239.255.255.250:1900\r\n\r\n"
	_, err = clientConn.Write([]byte(req))
	if err != nil {
		t.Fatalf("client write failed: %v", err)
	}

	resp := make([]byte, 2048)
	if err := clientConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline failed: %v", err)
	}
	n, err := clientConn.Read(resp)
	if err != nil {
		t.Fatalf("client read failed: %v", err)
	}

	if !bytes.Contains(resp[:n], []byte("HTTP/1.1 200")) {
		t.Errorf("expected HTTP 200 response, got: %s", string(resp[:n]))
	}

	if handler.count.Load() != 1 {
		t.Errorf("handler called %d times, want 1", handler.count.Load())
	}

	_ = s.Close()
}

type mockHandler struct {
	count atomic.Int32
}

func (h *mockHandler) Handle(req *http.Request) []Response {
	h.count.Add(1)
	rsp := make(Response)
	rsp["ST"] = "upnp:rootdevice"
	return []Response{rsp}
}
