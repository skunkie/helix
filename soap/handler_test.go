// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package soap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDetectPrefix(t *testing.T) {
	tests := []struct {
		name       string
		xml        string
		action     string
		wantPrefix string
	}{
		{
			name:       "prefixed with xmlns attribute and space",
			xml:        `<u:GetProtocolInfo xmlns:u="urn:schemas-upnp-org:service:ConnectionManager:1" />`,
			action:     "GetProtocolInfo",
			wantPrefix: "u",
		},
		{
			name:       "prefixed compact self-closing without space",
			xml:        `<u:GetProtocolInfo/>`,
			action:     "GetProtocolInfo",
			wantPrefix: "u",
		},
		{
			name:       "prefixed with open and close tag",
			xml:        `<m:Browse xmlns:m="urn:schemas-upnp-org:service:ContentDirectory:1"><ObjectID>0</ObjectID></m:Browse>`,
			action:     "Browse",
			wantPrefix: "m",
		},
		{
			name:       "prefixed with newline",
			xml:        "<u:IsAuthorized\nxmlns:u=\"urn:microsoft.com:service:X_MS_MediaReceiverRegistrar:1\"><DeviceID>1</DeviceID></u:IsAuthorized>",
			action:     "IsAuthorized",
			wantPrefix: "u",
		},
		{
			name:       "custom prefix with hyphens and digits",
			xml:        `<ns-1:Search xmlns:ns-1="urn:schemas-upnp-org:service:ContentDirectory:1"/>`,
			action:     "Search",
			wantPrefix: "ns-1",
		},
		{
			name:       "unprefixed tag with attributes",
			xml:        `<GetProtocolInfo xmlns="urn:schemas-upnp-org:service:ConnectionManager:1"/>`,
			action:     "GetProtocolInfo",
			wantPrefix: "",
		},
		{
			name:       "unprefixed simple tag",
			xml:        `<GetProtocolInfo></GetProtocolInfo>`,
			action:     "GetProtocolInfo",
			wantPrefix: "",
		},
		{
			name:       "non-matching action name",
			xml:        `<u:GetProtocolInfo xmlns:u="..."/>`,
			action:     "Browse",
			wantPrefix: "",
		},
		{
			name:       "empty XML",
			xml:        ``,
			action:     "GetProtocolInfo",
			wantPrefix: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectPrefix([]byte(tt.xml), tt.action)
			if got != tt.wantPrefix {
				t.Errorf("DetectPrefix(%q, %q) = %q, want %q", tt.xml, tt.action, got, tt.wantPrefix)
			}
		})
	}
}

type mockSOAPHandler struct {
	callFunc func(ctx context.Context, ns, action string, in []byte) ([]byte, error)
}

func (m *mockSOAPHandler) Call(ctx context.Context, ns, action string, in []byte) ([]byte, error) {
	return m.callFunc(ctx, ns, action, in)
}

type customSOAPError struct {
	code   FaultCode
	msg    string
	detail string
}

func (e *customSOAPError) Error() string        { return e.msg }
func (e *customSOAPError) FaultCode() FaultCode { return e.code }
func (e *customSOAPError) FaultString() string  { return e.msg }
func (e *customSOAPError) Detail() string       { return e.detail }

func TestHandle(t *testing.T) {
	validBody := `<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
  <s:Body>
    <u:GetProtocolInfo xmlns:u="urn:schemas-upnp-org:service:ConnectionManager:1"/>
  </s:Body>
</s:Envelope>`

	tests := []struct {
		name           string
		soapAction     string
		body           string
		handler        Interface
		wantStatusCode int
		wantSubstring  string
	}{
		{
			name:           "missing SOAPAction header",
			soapAction:     "",
			body:           validBody,
			handler:        &mockSOAPHandler{},
			wantStatusCode: http.StatusBadRequest,
			wantSubstring:  "must set SOAPAction header",
		},
		{
			name:           "malformed SOAPAction header without separator",
			soapAction:     `"urn:schemas-upnp-org:service:ConnectionManager:1"`,
			body:           validBody,
			handler:        &mockSOAPHandler{},
			wantStatusCode: http.StatusBadRequest,
			wantSubstring:  `SOAPAction header must be of form`,
		},
		{
			name:           "malformed XML request body",
			soapAction:     `"urn:schemas-upnp-org:service:ConnectionManager:1#GetProtocolInfo"`,
			body:           `not valid XML <><>`,
			handler:        &mockSOAPHandler{},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name:       "successful SOAP request",
			soapAction: `"urn:schemas-upnp-org:service:ConnectionManager:1#GetProtocolInfo"`,
			body:       validBody,
			handler: &mockSOAPHandler{
				callFunc: func(ctx context.Context, ns, action string, in []byte) ([]byte, error) {
					if ns != "urn:schemas-upnp-org:service:ConnectionManager:1" || action != "GetProtocolInfo" {
						t.Errorf("unexpected ns/action: %s/%s", ns, action)
					}
					return []byte("<result>OK</result>"), nil
				},
			},
			wantStatusCode: http.StatusOK,
			wantSubstring:  "<result>OK</result>",
		},
		{
			name:       "handler returns client SOAP fault error",
			soapAction: `"urn:schemas-upnp-org:service:ConnectionManager:1#GetProtocolInfo"`,
			body:       validBody,
			handler: &mockSOAPHandler{
				callFunc: func(ctx context.Context, ns, action string, in []byte) ([]byte, error) {
					return nil, &customSOAPError{code: FaultClient, msg: "Invalid Args", detail: "402"}
				},
			},
			wantStatusCode: http.StatusBadRequest,
			wantSubstring:  "<s:faultcode>s:Client</s:faultcode>",
		},
		{
			name:       "handler returns general internal error",
			soapAction: `"urn:schemas-upnp-org:service:ConnectionManager:1#GetProtocolInfo"`,
			body:       validBody,
			handler: &mockSOAPHandler{
				callFunc: func(ctx context.Context, ns, action string, in []byte) ([]byte, error) {
					return nil, errors.New("database connection failed")
				},
			},
			wantStatusCode: http.StatusInternalServerError,
			wantSubstring:  "<s:faultcode>s:Server</s:faultcode>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/control", strings.NewReader(tt.body))
			if tt.soapAction != "" {
				req.Header.Set("SOAPAction", tt.soapAction)
			}
			rec := httptest.NewRecorder()

			Handle(rec, req, tt.handler)

			if rec.Code != tt.wantStatusCode {
				t.Errorf("Handle() status = %d, want %d", rec.Code, tt.wantStatusCode)
			}
			if tt.wantSubstring != "" && !strings.Contains(rec.Body.String(), tt.wantSubstring) {
				t.Errorf("Handle() body %q does not contain %q", rec.Body.String(), tt.wantSubstring)
			}
		})
	}
}
