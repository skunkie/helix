// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

package upnp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestNewDeviceCache verifies that the device cache can discover devices
// and that its background refresh loop is properly canceled via context.
func TestNewDeviceCache(t *testing.T) {
	const (
		testURN  = URN("urn:schemas-upnp-org:device:test-cache:1")
		testUDN  = "uuid:test-cache-device"
		httpAddr = "127.0.0.1:19091"
		location = "http://" + httpAddr + "/"
	)

	// mockDevice encapsulates a mock UPnP device (SSDP and HTTP description).
	type mockDevice struct {
		ssdp   *mockSSDP
		http   *mockHTTP
		stop   func()
		closed bool
		mu     sync.Mutex
	}
	startMockDevice := func() *mockDevice {
		ssdp := newMockSSDP(t, testURN, testUDN, location)
		http := newMockHTTP(t, httpAddr, testURN, testUDN)
		return &mockDevice{
			ssdp: ssdp,
			http: http,
			stop: func() {
				ssdp.Close()
				http.Close()
			},
		}
	}

	mock := startMockDevice()

	ctx, cancel := context.WithCancel(context.Background())
	opts := DeviceCacheOptions{
		InitialRefresh: 100 * time.Millisecond,
		StableRefresh:  200 * time.Millisecond,
	}

	// Create a new DeviceCache, which will start discovering.
	cache := NewDeviceCache(ctx, testURN, opts)

	// Wait for the initial refresh to complete.
	time.Sleep(opts.InitialRefresh + 50*time.Millisecond)

	// The device should now be in the cache.
	if devices := cache.Devices(); len(devices) != 1 {
		t.Fatalf("wanted 1 device in cache, got %d", len(devices))
	}
	if _, ok := cache.DeviceByUDN(testUDN); !ok {
		t.Fatal("DeviceByUDN could not find the test device")
	}

	// Now, cancel the context to stop the refresh loop.
	cancel()

	// Stop the mock device so it's no longer discoverable.
	mock.stop()

	// Wait for a period longer than the refresh interval.
	time.Sleep(opts.StableRefresh + 50*time.Millisecond)

	// The device should *still* be in the cache because the refresh loop
	// should have been terminated by the context cancellation before it had
	// a chance to notice the device was gone.
	if len(cache.Devices()) != 1 {
		t.Fatal("device cache was cleared after context cancellation")
	}
}

// mockSSDP responds to M-SEARCH requests.
type mockSSDP struct {
	t        *testing.T
	conn     *net.UDPConn
	stopOnce sync.Once
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

func newMockSSDP(t *testing.T, urn URN, udn, location string) *mockSSDP {
	addr, err := net.ResolveUDPAddr("udp4", "239.255.255.250:1900")
	if err != nil {
		t.Fatalf("could not resolve SSDP address: %v", err)
	}
	conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	if err != nil {
		t.Skipf("could not listen on multicast UDP: %v (multicast may not be available in this environment)", err)
	}

	m := &mockSSDP{
		t:      t,
		conn:   conn,
		stopCh: make(chan struct{}),
	}

	m.wg.Add(1)
	go m.run(urn, udn, location)
	return m
}

func (m *mockSSDP) run(urn URN, udn, location string) {
	defer m.wg.Done()
	buf := make([]byte, 1024)
	for {
		select {
		case <-m.stopCh:
			return
		default:
		}

		m.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		n, from, err := m.conn.ReadFrom(buf)
		if err != nil {
			continue // Most likely a timeout
		}

		req := string(buf[:n])
		st := getHeader(req, "ST")
		if !strings.Contains(req, "M-SEARCH") || (st != string(urn) && st != "ssdp:all") {
			continue
		}

		resp := fmt.Sprintf(
			"HTTP/1.1 200 OK\r\nLOCATION: %s\r\nSERVER: test-server\r\nNT: %s\r\nUSN: %s::%s\r\nST: %s\r\n\r\n",
			location, urn, udn, urn, urn,
		)
		c, err := net.Dial("udp", from.String())
		if err != nil {
			continue
		}
		c.Write([]byte(resp))
		c.Close()
	}
}

func (m *mockSSDP) Close() {
	m.stopOnce.Do(func() {
		close(m.stopCh)
		m.conn.Close()
		m.wg.Wait()
	})
}

// mockHTTP serves a device description.
type mockHTTP struct {
	server *http.Server
}

func newMockHTTP(t *testing.T, addr string, urn URN, udn string) *mockHTTP {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<root xmlns="urn:schemas-upnp-org:device-1-0"><device><deviceType>%s</deviceType><UDN>%s</UDN></device></root>`, urn, udn)
	})
	server := &http.Server{Addr: addr, Handler: mux}

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			t.Errorf("mock HTTP server failed: %v", err)
		}
	}()

	return &mockHTTP{server: server}
}

func (m *mockHTTP) Close() {
	m.server.Shutdown(context.Background())
}

func getHeader(msg, header string) string {
	for _, line := range strings.Split(msg, "\r\n") {
		if strings.HasPrefix(strings.ToUpper(line), strings.ToUpper(header)+":") {
			return strings.TrimSpace(line[len(header)+1:])
		}
	}
	return ""
}
