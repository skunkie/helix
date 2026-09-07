// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package avtransport

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"testing"
	"time"

	"github.com/ethulhu/helix/upnpav"
)

type soapClientFunc func(context.Context, string, string, []byte) ([]byte, error)

func (f soapClientFunc) Call(ctx context.Context, namespace, action string, input []byte) ([]byte, error) {
	return f(ctx, namespace, action, input)
}

func TestClientControlRequests(t *testing.T) {
	tests := []struct {
		name    string
		action  string
		request interface{}
		call    func(Interface, context.Context) error
	}{
		{
			name:    "play",
			action:  play,
			request: playRequest{InstanceID: 0, Speed: "1"},
			call:    func(client Interface, ctx context.Context) error { return client.Play(ctx) },
		},
		{
			name:    "pause",
			action:  pause,
			request: pauseRequest{InstanceID: 0},
			call:    func(client Interface, ctx context.Context) error { return client.Pause(ctx) },
		},
		{
			name:    "next",
			action:  next,
			request: nextRequest{InstanceID: 0},
			call:    func(client Interface, ctx context.Context) error { return client.Next(ctx) },
		},
		{
			name:    "previous",
			action:  previous,
			request: previousRequest{InstanceID: 0},
			call:    func(client Interface, ctx context.Context) error { return client.Previous(ctx) },
		},
		{
			name:    "stop",
			action:  stop,
			request: stopRequest{InstanceID: 0},
			call:    func(client Interface, ctx context.Context) error { return client.Stop(ctx) },
		},
		{
			name:   "seek",
			action: seek,
			request: seekRequest{
				InstanceID: 0,
				Unit:       SeekRelativeTime,
				Target:     "0:01:02.500",
			},
			call: func(client Interface, ctx context.Context) error {
				return client.Seek(ctx, time.Minute+2*time.Second+500*time.Millisecond)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertClientRequest(t, test.action, test.request, test.call)
		})
	}
}

func TestClientSetURIRequests(t *testing.T) {
	uri := "https://example.com/audio.mp3"
	metadata, err := upnpav.DIDLForURI(uri)
	if err != nil {
		t.Fatalf("DIDLForURI returned error: %v", err)
	}

	t.Run("current URI with metadata", func(t *testing.T) {
		request := setAVTransportURIRequest{
			InstanceID:      0,
			CurrentURI:      uri,
			CurrentMetadata: upnpav.EncodedDIDLLite{DIDLLite: *metadata},
		}
		assertClientRequest(t, setAVTransportURI, request, func(client Interface, ctx context.Context) error {
			return client.SetCurrentURI(ctx, uri, metadata)
		})
	})

	t.Run("next URI with generated metadata", func(t *testing.T) {
		request := setNextAVTransportURIRequest{
			InstanceID:   0,
			NextURI:      uri,
			NextMetadata: upnpav.EncodedDIDLLite{DIDLLite: *metadata},
		}
		assertClientRequest(t, setNextAVTransportURI, request, func(client Interface, ctx context.Context) error {
			return client.SetNextURI(ctx, uri, nil)
		})
	})
}

func TestClientSetURIRejectsUnknownMediaType(t *testing.T) {
	called := false
	client := NewClient(soapClientFunc(func(context.Context, string, string, []byte) ([]byte, error) {
		called = true
		return nil, nil
	}))

	err := client.SetCurrentURI(context.Background(), "https://example.com/file.unknown", nil)
	if err == nil {
		t.Fatal("SetCurrentURI accepted an unknown media type")
	}
	if called {
		t.Fatal("SetCurrentURI called SOAP after metadata generation failed")
	}
}

func TestClientMediaInfoDecodesResponse(t *testing.T) {
	currentMetadata, err := upnpav.DIDLForURI("https://example.com/current.mp3")
	if err != nil {
		t.Fatalf("DIDLForURI returned error: %v", err)
	}
	nextMetadata, err := upnpav.DIDLForURI("https://example.com/next.mp3")
	if err != nil {
		t.Fatalf("DIDLForURI returned error: %v", err)
	}
	response := marshalResponse(t, getMediaInfoResponse{
		CurrentURI:      "https://example.com/current.mp3",
		CurrentMetadata: upnpav.EncodedDIDLLite{DIDLLite: *currentMetadata},
		NextURI:         "https://example.com/next.mp3",
		NextMetadata:    upnpav.EncodedDIDLLite{DIDLLite: *nextMetadata},
	})
	client := responseClient(t, getMediaInfo, response)

	currentURI, current, nextURI, next, err := client.MediaInfo(context.Background())
	if err != nil {
		t.Fatalf("MediaInfo returned error: %v", err)
	}
	if currentURI != "https://example.com/current.mp3" || nextURI != "https://example.com/next.mp3" {
		t.Fatalf("MediaInfo returned URIs %q and %q", currentURI, nextURI)
	}
	if len(current.Items) != 1 || current.Items[0].Title != currentURI {
		t.Fatalf("MediaInfo returned current metadata %#v", current)
	}
	if len(next.Items) != 1 || next.Items[0].Title != nextURI {
		t.Fatalf("MediaInfo returned next metadata %#v", next)
	}
}

func TestClientPositionInfoDecodesResponse(t *testing.T) {
	metadata, err := upnpav.DIDLForURI("https://example.com/current.mp3")
	if err != nil {
		t.Fatalf("DIDLForURI returned error: %v", err)
	}
	response := marshalResponse(t, getPositionInfoResponse{
		Duration:     upnpav.Duration{Duration: 5*time.Minute + 250*time.Millisecond},
		Metadata:     upnpav.EncodedDIDLLite{DIDLLite: *metadata},
		URI:          "https://example.com/current.mp3",
		RelativeTime: upnpav.Duration{Duration: 30*time.Second + 500*time.Millisecond},
	})
	client := responseClient(t, getPositionInfo, response)

	uri, gotMetadata, duration, elapsed, err := client.PositionInfo(context.Background())
	if err != nil {
		t.Fatalf("PositionInfo returned error: %v", err)
	}
	if uri != "https://example.com/current.mp3" {
		t.Fatalf("PositionInfo returned URI %q", uri)
	}
	if len(gotMetadata.Items) != 1 || gotMetadata.Items[0].Title != uri {
		t.Fatalf("PositionInfo returned metadata %#v", gotMetadata)
	}
	if duration != 5*time.Minute+250*time.Millisecond {
		t.Fatalf("PositionInfo returned duration %v", duration)
	}
	if elapsed != 30*time.Second+500*time.Millisecond {
		t.Fatalf("PositionInfo returned elapsed time %v", elapsed)
	}
}

func TestClientTransportInfoDecodesResponse(t *testing.T) {
	response := marshalResponse(t, getTransportInfoResponse{
		State:  StatePlaying,
		Status: StatusOK,
		Speed:  "1",
	})
	client := responseClient(t, getTransportInfo, response)

	state, status, err := client.TransportInfo(context.Background())
	if err != nil {
		t.Fatalf("TransportInfo returned error: %v", err)
	}
	if state != StatePlaying || status != StatusOK {
		t.Fatalf("TransportInfo returned %q, %q", state, status)
	}
}

func TestClientMapsUPnPError(t *testing.T) {
	client := NewClient(soapClientFunc(func(context.Context, string, string, []byte) ([]byte, error) {
		return nil, upnpav.ErrInvalidArgs
	}))

	err := client.Play(context.Background())
	var upnpError upnpav.Error
	if !errors.As(err, &upnpError) {
		t.Fatalf("Play returned %T, want upnpav.Error", err)
	}
	if upnpError.Code != upnpav.ErrInvalidArgs.Code ||
		upnpError.Description != upnpav.ErrInvalidArgs.Description {
		t.Fatalf("Play returned %v, want %v", upnpError, upnpav.ErrInvalidArgs)
	}
}

func TestClientRejectsMalformedResponse(t *testing.T) {
	client := responseClient(t, getPositionInfo, []byte("<invalid"))
	if _, _, _, _, err := client.PositionInfo(context.Background()); err == nil {
		t.Fatal("PositionInfo accepted malformed XML")
	}
}

func assertClientRequest(
	t *testing.T,
	action string,
	wantRequest interface{},
	call func(Interface, context.Context) error,
) {
	t.Helper()
	wantXML, err := xml.Marshal(wantRequest)
	if err != nil {
		t.Fatalf("could not marshal expected request: %v", err)
	}

	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "request-context")
	called := false
	client := NewClient(soapClientFunc(func(gotContext context.Context, namespace, gotAction string, input []byte) ([]byte, error) {
		called = true
		if gotContext != ctx {
			t.Error("client did not propagate request context")
		}
		if namespace != string(Version1) {
			t.Errorf("namespace = %q, want %q", namespace, Version1)
		}
		if gotAction != action {
			t.Errorf("action = %q, want %q", gotAction, action)
		}
		if !bytes.Equal(input, wantXML) {
			t.Errorf("request = %s, want %s", input, wantXML)
		}
		return nil, nil
	}))

	if err := call(client, ctx); err != nil {
		t.Fatalf("client call returned error: %v", err)
	}
	if !called {
		t.Fatal("client did not make SOAP call")
	}
}

func responseClient(t *testing.T, action string, response []byte) Interface {
	t.Helper()
	return NewClient(soapClientFunc(func(_ context.Context, namespace, gotAction string, _ []byte) ([]byte, error) {
		if namespace != string(Version1) {
			t.Errorf("namespace = %q, want %q", namespace, Version1)
		}
		if gotAction != action {
			t.Errorf("action = %q, want %q", gotAction, action)
		}
		return response, nil
	}))
}

func marshalResponse(t *testing.T, response interface{}) []byte {
	t.Helper()
	data, err := xml.Marshal(response)
	if err != nil {
		t.Fatalf("could not marshal response: %v", err)
	}
	return data
}
