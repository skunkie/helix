// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package netutil

import (
	"net"
	"testing"
)

func TestSuitableIPRequiresPrivateAddressForAutomaticSelection(t *testing.T) {
	tests := []struct {
		name string
		ip   string
	}{
		{name: "10/8", ip: "10.0.0.1"},
		{name: "172.16/12 lower bound", ip: "172.16.0.1"},
		{name: "172.16/12 upper bound", ip: "172.31.255.254"},
		{name: "192.168/16", ip: "192.168.0.1"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ip := net.ParseIP(test.ip)
			got, err := suitableIP([]net.Addr{&net.IPNet{IP: ip}}, true)
			if err != nil {
				t.Fatalf("suitableIP(%v) returned error: %v", ip, err)
			}
			if !got.Equal(ip) {
				t.Fatalf("suitableIP(%v) = %v", ip, got)
			}
		})
	}
}

func TestSuitableIPRejectsPublicAddressForAutomaticSelection(t *testing.T) {
	ip := net.ParseIP("203.0.113.1")
	if _, err := suitableIP([]net.Addr{&net.IPNet{IP: ip}}, true); err == nil {
		t.Fatal("suitableIP accepted a public address for automatic selection")
	}
}

func TestSuitableIPAcceptsExplicitPublicAddress(t *testing.T) {
	ip := net.ParseIP("203.0.113.1")
	got, err := suitableIP([]net.Addr{&net.IPNet{IP: ip}}, false)
	if err != nil {
		t.Fatalf("suitableIP(%v) returned error: %v", ip, err)
	}
	if !got.Equal(ip) {
		t.Fatalf("suitableIP(%v) = %v", ip, got)
	}
}
