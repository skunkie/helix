// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package upnp

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ethulhu/helix/upnp/scpd"
	"github.com/ethulhu/helix/xmltypes"
)

type eventSOAP struct{}

func (eventSOAP) Call(_ context.Context, _, action string, _ []byte) ([]byte, error) {
	switch action {
	case "GetSystemUpdateID":
		return []byte(`<GetSystemUpdateIDResponse><Id>42</Id></GetSystemUpdateIDResponse>`), nil
	case "GetProtocolInfo":
		return []byte(`<GetProtocolInfoResponse><Source>http-get:*:audio/mpeg:*</Source><Sink></Sink></GetProtocolInfoResponse>`), nil
	case "GetCurrentConnectionIDs":
		return []byte(`<GetCurrentConnectionIDsResponse><ConnectionIDs>0,7</ConnectionIDs></GetCurrentConnectionIDsResponse>`), nil
	default:
		return nil, errors.New("unsupported action")
	}
}

func TestParseCallbackURLs(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{
			input: "<http://192.168.1.193:2869/upnp/eventing/qrdjysnedl>",
			want:  []string{"http://192.168.1.193:2869/upnp/eventing/qrdjysnedl"},
		},
		{
			input: "<http://192.168.1.1:80/cb1> <http://192.168.1.2:80/cb2>",
			want:  []string{"http://192.168.1.1:80/cb1", "http://192.168.1.2:80/cb2"},
		},
		{
			input: "invalid",
			want:  nil,
		},
	}

	for i, tt := range tests {
		got := parseCallbackURLs(tt.input)
		if len(got) != len(tt.want) {
			t.Fatalf("[%d]: got %d URLs, want %d", i, len(got), len(tt.want))
		}
		for j, u := range got {
			if u.String() != tt.want[j] {
				t.Errorf("[%d][%d]: got %s, want %s", i, j, u.String(), tt.want[j])
			}
		}
	}
}

func TestEventing_SubscribeAndNotify(t *testing.T) {
	type notification struct {
		body string
		seq  string
	}
	notifyReceived := make(chan notification, 2)
	notifyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "NOTIFY" {
			t.Errorf("expected NOTIFY method, got %s", r.Method)
		}
		if r.Header.Get("NT") != "upnp:event" {
			t.Errorf("expected NT=upnp:event, got %s", r.Header.Get("NT"))
		}
		if r.Header.Get("NTS") != "upnp:propchange" {
			t.Errorf("expected NTS=upnp:propchange, got %s", r.Header.Get("NTS"))
		}
		body, _ := io.ReadAll(r.Body)
		notifyReceived <- notification{body: string(body), seq: r.Header.Get("SEQ")}
		w.WriteHeader(http.StatusOK)
	}))
	defer notifyServer.Close()

	d := &Device{
		Name:      "test",
		ModelName: "Helix",
	}
	const serviceURN = URN("urn:schemas-upnp-org:service:ContentDirectory:1")
	d.Handle(serviceURN, "ContentDirectory", scpd.Document{StateVariables: []scpd.StateVariable{{
		SendEventsAttribute: xmltypes.Yes,
		Name:                "SystemUpdateID",
		DataType:            "ui4",
	}}}, eventSOAP{})

	handler := d.HTTPHandler("/upnp/")

	// 1. Initial Subscribe
	req := httptest.NewRequest("SUBSCRIBE", "/urn:schemas-upnp-org:service:ContentDirectory:1", nil)
	req.Header.Set("Callback", "<"+notifyServer.URL+">")
	req.Header.Set("NT", "upnp:event")
	req.Header.Set("Timeout", "Second-1800")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	sid := rec.Header().Get("SID")
	if sid == "" {
		t.Fatal("expected non-empty SID header")
	}
	if rec.Header().Get("TIMEOUT") != "Second-1800" {
		t.Errorf("expected TIMEOUT=Second-1800, got %s", rec.Header().Get("TIMEOUT"))
	}

	select {
	case notification := <-notifyReceived:
		if notification.seq != "0" {
			t.Errorf("initial sequence = %q, want 0", notification.seq)
		}
		if !strings.Contains(notification.body, "<SystemUpdateID>42</SystemUpdateID>") {
			t.Errorf("initial event does not contain current state: %s", notification.body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for initial event NOTIFY")
	}

	// 2. Renewal Subscribe
	renewReq := httptest.NewRequest("SUBSCRIBE", "/urn:schemas-upnp-org:service:ContentDirectory:1", nil)
	renewReq.Header.Set("SID", sid)
	renewReq.Header.Set("Timeout", "Second-1800")

	renewRec := httptest.NewRecorder()
	handler.ServeHTTP(renewRec, renewReq)

	if renewRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for renewal, got %d", renewRec.Code)
	}
	if renewRec.Header().Get("SID") != sid {
		t.Errorf("expected SID=%s, got %s", sid, renewRec.Header().Get("SID"))
	}

	if err := d.NotifySubscribers(context.Background(), serviceURN); err != nil {
		t.Fatalf("NotifySubscribers failed: %v", err)
	}
	select {
	case notification := <-notifyReceived:
		if notification.seq != "1" {
			t.Errorf("update sequence = %q, want 1", notification.seq)
		}
		if !strings.Contains(notification.body, "<SystemUpdateID>42</SystemUpdateID>") {
			t.Errorf("update does not contain current state: %s", notification.body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for update NOTIFY")
	}

	// 3. Unsubscribe
	unsubReq := httptest.NewRequest("UNSUBSCRIBE", "/urn:schemas-upnp-org:service:ContentDirectory:1", nil)
	unsubReq.Header.Set("SID", sid)

	unsubRec := httptest.NewRecorder()
	handler.ServeHTTP(unsubRec, unsubReq)

	if unsubRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for unsubscribe, got %d", unsubRec.Code)
	}
	if len(d.subscriptions) != 0 {
		t.Fatalf("subscriptions remain after unsubscribe: %v", d.subscriptions)
	}

	unknownRenewal := httptest.NewRequest("SUBSCRIBE", "/urn:schemas-upnp-org:service:ContentDirectory:1", nil)
	unknownRenewal.Header.Set("SID", sid)
	unknownRecorder := httptest.NewRecorder()
	handler.ServeHTTP(unknownRecorder, unknownRenewal)
	if unknownRecorder.Code != http.StatusPreconditionFailed {
		t.Errorf("unknown renewal status = %d, want %d", unknownRecorder.Code, http.StatusPreconditionFailed)
	}
}

func TestEventPropertiesAreServiceSpecific(t *testing.T) {
	const urn = URN("urn:schemas-upnp-org:service:ConnectionManager:1")
	d := &Device{}
	d.Handle(urn, "ConnectionManager", scpd.Document{StateVariables: []scpd.StateVariable{
		{SendEventsAttribute: xmltypes.Yes, Name: "SourceProtocolInfo", DataType: "string"},
		{SendEventsAttribute: xmltypes.Yes, Name: "SinkProtocolInfo", DataType: "string"},
		{SendEventsAttribute: xmltypes.Yes, Name: "CurrentConnectionIDs", DataType: "string"},
		{SendEventsAttribute: xmltypes.No, Name: "A_ARG_TYPE_ConnectionID", DataType: "i4"},
	}}, eventSOAP{})

	properties := d.eventProperties(context.Background(), urn)
	want := []property{
		{Name: "SourceProtocolInfo", Value: "http-get:*:audio/mpeg:*"},
		{Name: "SinkProtocolInfo", Value: ""},
		{Name: "CurrentConnectionIDs", Value: "0,7"},
	}
	if !reflect.DeepEqual(properties, want) {
		t.Fatalf("properties = %#v, want %#v", properties, want)
	}
}
