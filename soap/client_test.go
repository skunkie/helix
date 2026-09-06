// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package soap

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type closeTrackingBody struct {
	io.Reader
	closed bool
}

func (b *closeTrackingBody) Close() error {
	b.closed = true
	return nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClientClosesResponseBody(t *testing.T) {
	body := &closeTrackingBody{Reader: strings.NewReader(`<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/"><s:Body><Result>ok</Result></s:Body></s:Envelope>`)}
	previousClient := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: body}, nil
	})}
	defer func() { http.DefaultClient = previousClient }()

	baseURL, err := url.Parse("http://device.invalid/control")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewClient(baseURL).Call(context.Background(), "urn:test", "Test", nil); err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	if !body.closed {
		t.Fatal("response body was not closed")
	}
}
