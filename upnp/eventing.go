// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package upnp

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ethulhu/helix/logger"
)

var callbackURLRegexp = regexp.MustCompile(`<([^>]+)>`)

// parseCallbackURLs parses URLs in angle brackets from the CALLBACK header.
func parseCallbackURLs(callback string) []*url.URL {
	var urls []*url.URL
	for _, match := range callbackURLRegexp.FindAllStringSubmatch(callback, -1) {
		if u, err := url.Parse(match[1]); err == nil {
			urls = append(urls, u)
		}
	}
	return urls
}

func generateSID() string {
	var b [16]byte
	_, _ = io.ReadFull(rand.Reader, b[:])
	return fmt.Sprintf("uuid:%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

type propertySet struct {
	XMLName    xml.Name   `xml:"e:propertyset"`
	Xmlns      string     `xml:"xmlns:e,attr"`
	Properties []property `xml:"e:property"`
}

type property struct {
	Name  string `xml:"-"`
	Value string `xml:"-"`
}

func (p property) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "e:property"
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	if err := e.EncodeElement(p.Value, xml.StartElement{Name: xml.Name{Local: p.Name}}); err != nil {
		return err
	}
	return e.EncodeToken(start.End())
}

type eventSubscription struct {
	URN       URN
	Callbacks []*url.URL
	ExpiresAt time.Time
	Sequence  uint32
	Ready     <-chan struct{}
}

func subscriptionTimeout(raw string) (time.Duration, string) {
	if strings.EqualFold(strings.TrimSpace(raw), "Second-infinite") {
		return 0, "Second-infinite"
	}

	seconds := uint64(1800)
	raw = strings.TrimSpace(raw)
	if len(raw) > len("Second-") && strings.EqualFold(raw[:len("Second-")], "Second-") {
		parsed, err := strconv.ParseUint(raw[len("Second-"):], 10, 64)
		maxSeconds := uint64((1<<63 - 1) / int64(time.Second))
		if err == nil && parsed > 0 && parsed <= maxSeconds {
			seconds = parsed
		}
	}
	return time.Duration(seconds) * time.Second, fmt.Sprintf("Second-%d", seconds) //nolint:gosec // seconds is bounded by time.Duration's maximum.
}

func subscriptionExpiry(timeout time.Duration) time.Time {
	if timeout == 0 {
		return time.Time{}
	}
	return time.Now().Add(timeout)
}

func (d *Device) handleSubscribe(w http.ResponseWriter, r *http.Request, urn URN) {
	log, _ := logger.FromContext(r.Context())
	sid := r.Header.Get("Sid")
	timeout, timeoutHeader := subscriptionTimeout(r.Header.Get("Timeout"))

	if sid == "" {
		if !strings.EqualFold(strings.TrimSpace(r.Header.Get("NT")), "upnp:event") {
			http.Error(w, "NT must be upnp:event", http.StatusPreconditionFailed)
			return
		}

		callbackHeader := r.Header.Get("Callback")
		urls := parseCallbackURLs(callbackHeader)
		if len(urls) == 0 {
			http.Error(w, "missing or invalid CALLBACK header", http.StatusBadRequest)
			return
		}
		for _, callback := range urls {
			if callback.Host == "" || (callback.Scheme != "http" && callback.Scheme != "https") {
				http.Error(w, "CALLBACK must contain an absolute HTTP URL", http.StatusPreconditionFailed)
				return
			}
		}

		newSID := generateSID()
		ready := make(chan struct{})
		d.mu.Lock()
		if d.subscriptions == nil {
			d.subscriptions = make(map[string]eventSubscription)
		}
		d.subscriptions[newSID] = eventSubscription{
			URN:       urn,
			Callbacks: urls,
			ExpiresAt: subscriptionExpiry(timeout),
			Ready:     ready,
		}
		d.mu.Unlock()

		w.Header().Set("SID", newSID)
		w.Header().Set("TIMEOUT", timeoutHeader)
		w.Header().Set("Server", fmt.Sprintf("DLNADOC/1.50 UPnP/1.0 %s/1", d.ModelName))
		w.WriteHeader(http.StatusOK)
		log.WithField("sid", newSID).WithField("callbacks", callbackHeader).Debug("accepted event subscription")

		eventContext := context.WithoutCancel(r.Context())
		go func() {
			defer close(ready)
			time.Sleep(50 * time.Millisecond)
			d.mu.RLock()
			_, subscribed := d.subscriptions[newSID]
			d.mu.RUnlock()
			if !subscribed {
				return
			}
			if err := d.sendEvent(eventContext, urn, urls, newSID, 0); err != nil {
				log.WithError(err).Warning("could not send initial event")
			}
		}()
		return
	}

	if r.Header.Get("Callback") != "" || r.Header.Get("NT") != "" {
		http.Error(w, "renewal must not include CALLBACK or NT", http.StatusBadRequest)
		return
	}

	d.mu.Lock()
	subscription, ok := d.subscriptions[sid]
	if ok && !subscription.ExpiresAt.IsZero() && time.Now().After(subscription.ExpiresAt) {
		delete(d.subscriptions, sid)
		ok = false
	}
	if ok && subscription.URN == urn {
		subscription.ExpiresAt = subscriptionExpiry(timeout)
		d.subscriptions[sid] = subscription
	} else {
		ok = false
	}
	d.mu.Unlock()
	if !ok {
		http.Error(w, "unknown subscription", http.StatusPreconditionFailed)
		return
	}

	w.Header().Set("SID", sid)
	w.Header().Set("TIMEOUT", timeoutHeader)
	w.Header().Set("Server", fmt.Sprintf("DLNADOC/1.50 UPnP/1.0 %s/1", d.ModelName))
	w.WriteHeader(http.StatusOK)
	log.WithField("sid", sid).Debug("renewed event subscription")
}

func (d *Device) handleUnsubscribe(w http.ResponseWriter, r *http.Request, urn URN) {
	log, _ := logger.FromContext(r.Context())
	sid := r.Header.Get("Sid")
	if sid == "" || r.Header.Get("Callback") != "" || r.Header.Get("NT") != "" {
		http.Error(w, "invalid unsubscribe request", http.StatusBadRequest)
		return
	}

	d.mu.Lock()
	subscription, ok := d.subscriptions[sid]
	if ok && subscription.URN == urn {
		delete(d.subscriptions, sid)
	} else {
		ok = false
	}
	d.mu.Unlock()
	if !ok {
		http.Error(w, "unknown subscription", http.StatusPreconditionFailed)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.WithField("sid", sid).Debug("unsubscribed from events")
}

func (d *Device) eventProperties(ctx context.Context, urn URN) []property {
	d.mu.RLock()
	service, ok := d.serviceByURN[urn]
	d.mu.RUnlock()
	if !ok {
		return nil
	}

	values := make(map[string]string)
	if service.SOAPInterface != nil {
		call := func(action string, output interface{}) bool {
			input := fmt.Appendf(nil, `<%s xmlns=%q/>`, action, urn)
			response, err := service.SOAPInterface.Call(ctx, string(urn), action, input)
			return err == nil && xml.Unmarshal(response, output) == nil //nolint:gosec // Service responses are trusted local UPnP XML and decoded without entity expansion.
		}

		var update struct {
			ID string `xml:"Id"`
		}
		if call("GetSystemUpdateID", &update) {
			values["SystemUpdateID"] = update.ID
		}

		var protocols struct {
			Source string `xml:"Source"`
			Sink   string `xml:"Sink"`
		}
		if call("GetProtocolInfo", &protocols) {
			values["SourceProtocolInfo"] = protocols.Source
			values["SinkProtocolInfo"] = protocols.Sink
		}

		var connections struct {
			IDs string `xml:"ConnectionIDs"`
		}
		if call("GetCurrentConnectionIDs", &connections) {
			values["CurrentConnectionIDs"] = connections.IDs
		}
	}

	properties := make([]property, 0)
	for _, variable := range service.SCPD.StateVariables {
		if !bool(variable.SendEventsAttribute) {
			continue
		}
		value, found := values[variable.Name]
		if !found {
			switch variable.DataType {
			case "ui1", "ui2", "ui4", "i1", "i2", "i4", "int", "float", "fixed.14.4", "number":
				value = "0"
			}
		}
		properties = append(properties, property{Name: variable.Name, Value: value})
	}
	return properties
}

func (d *Device) sendEvent(parent context.Context, urn URN, urls []*url.URL, sid string, sequence uint32) error {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()

	props := propertySet{
		Xmlns:      "urn:schemas-upnp-org:event-1-0",
		Properties: d.eventProperties(ctx, urn),
	}
	body, err := xml.Marshal(props)
	if err != nil {
		return err
	}
	body = append([]byte(xml.Header), body...)

	client := &http.Client{Timeout: 5 * time.Second}
	var errs []error
	for _, u := range urls {
		req, err := http.NewRequestWithContext(ctx, "NOTIFY", u.String(), bytes.NewReader(body))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		req.Header.Set("CONTENT-TYPE", `text/xml; charset="utf-8"`)
		req.Header.Set("NT", "upnp:event")
		req.Header.Set("NTS", "upnp:propchange")
		req.Header.Set("SID", sid)
		req.Header.Set("SEQ", fmt.Sprintf("%d", sequence))

		resp, err := client.Do(req)
		if err != nil {
			errs = append(errs, fmt.Errorf("notify %s: %w", u, err))
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			errs = append(errs, fmt.Errorf("notify %s: %s", u, resp.Status))
		}
	}
	return errors.Join(errs...)
}

// NotifySubscribers sends the current evented state for a service to all active subscribers.
func (d *Device) NotifySubscribers(ctx context.Context, urn URN) error {
	type delivery struct {
		sid       string
		callbacks []*url.URL
		sequence  uint32
		ready     <-chan struct{}
	}

	now := time.Now()
	d.mu.Lock()
	var deliveries []delivery
	for sid, subscription := range d.subscriptions {
		if !subscription.ExpiresAt.IsZero() && now.After(subscription.ExpiresAt) {
			delete(d.subscriptions, sid)
			continue
		}
		if subscription.URN != urn {
			continue
		}
		subscription.Sequence++
		d.subscriptions[sid] = subscription
		deliveries = append(deliveries, delivery{sid: sid, callbacks: subscription.Callbacks, sequence: subscription.Sequence, ready: subscription.Ready})
	}
	d.mu.Unlock()

	var errs []error
	for _, delivery := range deliveries {
		select {
		case <-delivery.ready:
		case <-ctx.Done():
			return errors.Join(append(errs, ctx.Err())...)
		}
		d.mu.RLock()
		subscription, subscribed := d.subscriptions[delivery.sid]
		d.mu.RUnlock()
		if !subscribed || subscription.URN != urn {
			continue
		}
		if err := d.sendEvent(ctx, urn, delivery.callbacks, delivery.sid, delivery.sequence); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
