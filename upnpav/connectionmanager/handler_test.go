// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package connectionmanager

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestConnectionManagerServerAndHandler(t *testing.T) {
	ctx := context.Background()
	server := NewServer(nil, nil)
	handler := SOAPHandler{Interface: server}
	client := NewClient(handler)

	// Test ProtocolInfo
	sources, sinks, err := client.ProtocolInfo(ctx)
	if err != nil {
		t.Fatalf("ProtocolInfo failed: %v", err)
	}
	if len(sources) == 0 {
		t.Errorf("expected non-empty default sources")
	}
	if len(sinks) != 0 {
		t.Errorf("expected empty sinks, got %v", sinks)
	}

	// Test CurrentConnectionIDs
	ids, err := client.CurrentConnectionIDs(ctx)
	if err != nil {
		t.Fatalf("CurrentConnectionIDs failed: %v", err)
	}
	if !reflect.DeepEqual(ids, []int{0}) {
		t.Errorf("CurrentConnectionIDs = %v, want [0]", ids)
	}

	// Test CurrentConnectionInfo for ID 0
	info, err := client.CurrentConnectionInfo(ctx, 0)
	if err != nil {
		t.Fatalf("CurrentConnectionInfo(0) failed: %v", err)
	}
	if info.Direction != Output || info.Status != StatusOK {
		t.Errorf("unexpected info: %+v", info)
	}

	// Test CurrentConnectionInfo for non-existent ID 99
	_, err = client.CurrentConnectionInfo(ctx, 99)
	if err == nil {
		t.Errorf("expected error for invalid connection ID")
	}
}

func TestSOAPNamespaceAndPrefix(t *testing.T) {
	ctx := context.Background()
	handler := SOAPHandler{Interface: NewServer(nil, nil)}

	// Prefixed SOAP request for GetProtocolInfo (compact self-closing tag without space)
	reqXML := []byte(`<u:GetProtocolInfo xmlns:u="urn:schemas-upnp-org:service:ConnectionManager:1"/>`)
	respBytes, err := handler.Call(ctx, string(Version1), "GetProtocolInfo", reqXML)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	if !strings.Contains(string(respBytes), "u:GetProtocolInfoResponse") {
		t.Errorf("expected echoed prefix 'u:', got: %s", string(respBytes))
	}

	// Prefixed SOAP request for GetCurrentConnectionIDs (open/close tag)
	reqXML = []byte(`<m:GetCurrentConnectionIDs xmlns:m="urn:schemas-upnp-org:service:ConnectionManager:1"></m:GetCurrentConnectionIDs>`)
	respBytes, err = handler.Call(ctx, string(Version1), "GetCurrentConnectionIDs", reqXML)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	if !strings.Contains(string(respBytes), "m:GetCurrentConnectionIDsResponse") {
		t.Errorf("expected echoed prefix 'm:', got: %s", string(respBytes))
	}

	// Unprefixed SOAP request for GetProtocolInfo
	reqXML = []byte(`<GetProtocolInfo xmlns="urn:schemas-upnp-org:service:ConnectionManager:1" />`)
	respBytes, err = handler.Call(ctx, string(Version1), "GetProtocolInfo", reqXML)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	if strings.Contains(string(respBytes), ":GetProtocolInfoResponse") {
		t.Errorf("expected no prefix in response, got: %s", string(respBytes))
	}

	// Invalid namespace
	_, err = handler.Call(ctx, "invalid-namespace", "GetProtocolInfo", reqXML)
	if err == nil {
		t.Errorf("expected error for invalid namespace")
	}

	// Invalid action
	_, err = handler.Call(ctx, string(Version1), "InvalidAction", reqXML)
	if err == nil {
		t.Errorf("expected error for invalid action")
	}

	// Invalid XML
	_, err = handler.Call(ctx, string(Version1), "GetCurrentConnectionInfo", []byte("bad-xml"))
	if err == nil {
		t.Errorf("expected error for bad XML")
	}
}
