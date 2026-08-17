// SPDX-FileCopyrightText: 2020 Ethel Morgan
// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package upnp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/ethulhu/helix/upnp/scpd"
	"github.com/ethulhu/helix/upnp/ssdp"
)

type mockSOAP struct{}

func (m *mockSOAP) Call(ctx context.Context, ns, action string, in []byte) ([]byte, error) {
	return []byte("<result>ok</result>"), nil
}

func TestDeviceManifest(t *testing.T) {
	tests := []struct {
		urns []URN
		ids  []ServiceID
		want ssdp.Document
	}{
		{
			urns: []URN{"hello"},
			ids:  []ServiceID{"goodbye"},
			want: ssdp.Document{
				NSDLNA:      "urn:schemas-dlna-org:device-1-0",
				NSSEC:       "http://www.sec.co.kr/dlna",
				SpecVersion: ssdp.Version,
				Device: ssdp.Device{
					FriendlyName: "name",
					UDN:          "udn",
					DLNACAP:      &ssdp.DLNACAP{},
					DLNADOC:      []string{"DMS-1.50", "M-DMS-1.50"},
					SecCap:       "smi,DCM10,getMediaInfo.sec,getCaptionInfo.sec",
					XSecCap:      "smi,DCM10,getMediaInfo.sec,getCaptionInfo.sec",
					ServiceList: ssdp.ServiceList{
						Services: []ssdp.Service{
							{
								ServiceType: "hello",
								ServiceID:   "goodbye",
								SCPDURL:     "/hello",
								ControlURL:  "/hello",
								EventSubURL: "/hello",
							},
						},
					},
				},
			},
		},
	}

	for i, tt := range tests {
		d := &Device{
			Name: "name",
			UDN:  "udn",
		}
		for i, urn := range tt.urns {
			d.Handle(urn, tt.ids[i], scpd.Document{}, nil)
		}

		got := d.manifest("/")
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("[%d]: got:\n\n%v\n\nwant:\n\n%v", i, got, tt.want)
		}
	}
}

func TestDeviceAccessors(t *testing.T) {
	d := &Device{Name: "TestDevice", UDN: "uuid:123"}
	const urn = URN("urn:schemas-upnp-org:service:ContentDirectory:1")
	const serviceID = ServiceID("urn:upnp-org:serviceId:ContentDirectory")
	soapHandler := &mockSOAP{}

	d.Handle(urn, serviceID, scpd.Document{SpecVersion: scpd.Version}, soapHandler)

	services := d.Services()
	if len(services) != 1 || services[0] != urn {
		t.Errorf("Services() = %v, want [%v]", services, urn)
	}

	si, ok := d.SOAPInterface(urn)
	if !ok || si != soapHandler {
		t.Errorf("SOAPInterface(%v) = %v, %v, want %v, true", urn, si, ok, soapHandler)
	}

	_, ok = d.SOAPInterface("urn:unknown")
	if ok {
		t.Errorf("SOAPInterface(unknown) expected false, got true")
	}

	if d.BootID() != 0 {
		t.Errorf("BootID() initial = %d, want 0", d.BootID())
	}
	d.SetBootID(42)
	if d.BootID() != 42 {
		t.Errorf("BootID() after Set = %d, want 42", d.BootID())
	}
	d.IncrementBootID()
	if d.BootID() != 43 {
		t.Errorf("BootID() after Increment = %d, want 43", d.BootID())
	}
}

func TestDeviceHTTPHandler(t *testing.T) {
	d := &Device{
		Name:      "MediaServer",
		ModelName: "Helix",
		UDN:       "uuid:device-1",
	}
	const serviceURN = URN("urn:schemas-upnp-org:service:ContentDirectory:1")
	const serviceID = ServiceID("urn:upnp-org:serviceId:ContentDirectory")
	doc := scpd.Document{SpecVersion: scpd.Version}
	soapHandler := &mockSOAP{}
	d.Handle(serviceURN, serviceID, doc, soapHandler)

	handler := d.HTTPHandler("/")

	t.Run("serve manifest on root description path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/description.xml", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if ct := rec.Header().Get("Content-Type"); ct != `text/xml; charset="utf-8"` {
			t.Errorf("Content-Type = %q, want text/xml; charset=\"utf-8\"", ct)
		}
		if !strings.Contains(rec.Body.String(), "MediaServer") {
			t.Errorf("body does not contain friendly name: %s", rec.Body.String())
		}
	})

	t.Run("serve SCPD document for service", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/urn:schemas-upnp-org:service:ContentDirectory:1", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if ct := rec.Header().Get("Content-Type"); ct != `text/xml; charset="utf-8"` {
			t.Errorf("Content-Type = %q, want text/xml; charset=\"utf-8\"", ct)
		}
		if !strings.Contains(rec.Body.String(), "<scpd") {
			t.Errorf("body does not contain scpd tag: %s", rec.Body.String())
		}
	})

	t.Run("serve SOAP POST action on service path", func(t *testing.T) {
		body := `<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
  <s:Body>
    <u:Browse xmlns:u="urn:schemas-upnp-org:service:ContentDirectory:1"/>
  </s:Body>
</s:Envelope>`
		req := httptest.NewRequest(http.MethodPost, "/urn:schemas-upnp-org:service:ContentDirectory:1", strings.NewReader(body))
		req.Header.Set("SOAPAction", `"urn:schemas-upnp-org:service:ContentDirectory:1#Browse"`)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if !strings.Contains(rec.Body.String(), "<result>ok</result>") {
			t.Errorf("body does not contain expected result: %s", rec.Body.String())
		}
	})

	t.Run("serve SOAP POST fallback routing via SOAPAction header", func(t *testing.T) {
		body := `<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
  <s:Body>
    <u:Browse xmlns:u="urn:schemas-upnp-org:service:ContentDirectory:1"/>
  </s:Body>
</s:Envelope>`
		req := httptest.NewRequest(http.MethodPost, "/control", strings.NewReader(body))
		req.Header.Set("SOAPAction", `"urn:schemas-upnp-org:service:ContentDirectory:1#Browse"`)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("unknown service returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/unknown-service-path", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("unsupported method returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/urn:schemas-upnp-org:service:ContentDirectory:1", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})
}

func TestIsManifestRequest(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/", true},
		{"", true},
		{"/description.xml", true},
		{"/rootDesc.xml", true},
		{"/desc.xml", true},
		{"/device.xml", true},
		{"/Description.XML", true},
		{"/ROOTDESC.XML", true},
		{"/description.xml/", true},
		{"/hello", false},
		{"/urn:schemas-upnp-org:service:ContentDirectory:1", false},
		{"/objects/file.mp4", false},
	}

	for _, tt := range tests {
		if got := isManifestRequest(tt.path); got != tt.want {
			t.Errorf("isManifestRequest(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
