// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package mediareceiverregistrar

import (
	"context"
	"encoding/xml"
	"strings"
	"testing"
)

func TestMediaReceiverRegistrarServerAndHandler(t *testing.T) {
	ctx := context.Background()
	server := NewServer()
	handler := SOAPHandler{Interface: server}
	client := NewClient(handler)

	// Test IsAuthorized
	res, err := client.IsAuthorized(ctx, "uuid:1234-5678")
	if err != nil {
		t.Fatalf("IsAuthorized failed: %v", err)
	}
	if res != 1 {
		t.Errorf("IsAuthorized = %d, want 1", res)
	}

	// Test IsValid
	res, err = client.IsValid(ctx, "uuid:1234-5678")
	if err != nil {
		t.Fatalf("IsValid failed: %v", err)
	}
	if res != 1 {
		t.Errorf("IsValid = %d, want 1", res)
	}

	// Test RegisterDevice
	respMsg, err := client.RegisterDevice(ctx, []byte("test-data"))
	if err != nil {
		t.Fatalf("RegisterDevice failed: %v", err)
	}
	if len(respMsg) != 0 {
		t.Errorf("RegisterDevice returned non-empty message: %v", respMsg)
	}
}

func TestSOAPNamespaceAndPrefix(t *testing.T) {
	ctx := context.Background()
	handler := SOAPHandler{Interface: NewServer()}

	// Prefixed SOAP request
	reqXML := []byte(`<u:IsAuthorized xmlns:u="urn:microsoft.com:service:X_MS_MediaReceiverRegistrar:1"><DeviceID>uuid:1234</DeviceID></u:IsAuthorized>`)
	respBytes, err := handler.Call(ctx, string(Version1), "IsAuthorized", reqXML)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	if !strings.Contains(string(respBytes), "u:IsAuthorizedResponse") {
		t.Errorf("expected echoed prefix 'u:', got: %s", string(respBytes))
	}

	// Prefixed SOAP request with custom prefix and closing tag
	reqXML = []byte(`<m:IsValid xmlns:m="urn:microsoft.com:service:X_MS_MediaReceiverRegistrar:1"><DeviceID>uuid:1234</DeviceID></m:IsValid>`)
	respBytes, err = handler.Call(ctx, string(Version1), "IsValid", reqXML)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	if !strings.Contains(string(respBytes), "m:IsValidResponse") {
		t.Errorf("expected echoed prefix 'm:', got: %s", string(respBytes))
	}

	// Unprefixed SOAP request
	reqXML = []byte(`<IsAuthorized xmlns="urn:microsoft.com:service:X_MS_MediaReceiverRegistrar:1"><DeviceID>uuid:1234</DeviceID></IsAuthorized>`)
	respBytes, err = handler.Call(ctx, string(Version1), "IsAuthorized", reqXML)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	if strings.Contains(string(respBytes), ":IsAuthorizedResponse") {
		t.Errorf("expected no prefix in response, got: %s", string(respBytes))
	}

	// Invalid namespace
	_, err = handler.Call(ctx, "invalid-namespace", "IsAuthorized", reqXML)
	if err == nil {
		t.Errorf("expected error for invalid namespace")
	}

	// Invalid action
	_, err = handler.Call(ctx, string(Version1), "InvalidAction", reqXML)
	if err == nil {
		t.Errorf("expected error for invalid action")
	}

	// Invalid XML
	_, err = handler.Call(ctx, string(Version1), "IsAuthorized", []byte("bad-xml"))
	if err == nil {
		t.Errorf("expected error for bad XML")
	}
}

func TestSCPD(t *testing.T) {
	bytes, err := xml.Marshal(SCPD)
	if err != nil {
		t.Fatalf("marshalling SCPD: %v", err)
	}
	if len(bytes) == 0 {
		t.Errorf("expected non-empty SCPD XML")
	}
}
