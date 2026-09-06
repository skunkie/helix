// SPDX-FileCopyrightText: 2020 Ethel Morgan
// SPDX-FileCopyrightText: 2025-2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package upnp

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ethulhu/helix/logger"
	"github.com/ethulhu/helix/upnp/httpu"
	"github.com/ethulhu/helix/upnp/ssdp"
	"golang.org/x/net/ipv4"
)

const (
	discoverMethod = "M-SEARCH"
	notifyMethod   = "NOTIFY"
	notifyAlive    = "ssdp:alive"
	notifyUpdate   = "ssdp:update"
	notifyByeBye   = "ssdp:byebye"

	ssdpCacheControl = "max-age=300"
)

var (
	ssdpBroadcastAddr = &net.UDPAddr{
		IP:   net.IPv4(239, 255, 255, 250),
		Port: 1900,
	}

	discoverURL = &url.URL{Opaque: "*"}
)

// DiscoverURLs discovers UPnP device manifest URLs using SSDP on the local network.
// It returns all valid URLs it finds, a slice of errors from invalid SSDP responses, and an error with the actual connection itself.
func DiscoverURLs(ctx context.Context, urn URN, iface *net.Interface) ([]*url.URL, []error, error) {
	req := discoverRequest(ctx, urn)

	rsps, errs, err := httpu.Do(req, 3, iface)

	locations := map[string]*url.URL{}
	for _, rsp := range rsps {
		location, err := rsp.Location()
		if err != nil {
			errs = append(errs, fmt.Errorf("could not find SSDP response Location: %w", err))
			continue
		}
		locations[location.String()] = location
	}

	var urls []*url.URL
	for _, location := range locations {
		urls = append(urls, location)
	}
	return urls, errs, err
}

// DiscoverDevices discovers UPnP devices using SSDP on the local network.
// It returns all valid URLs it finds, a slice of errors from invalid SSDP responses or UPnP device manifests, and an error with the actual connection itself.
func DiscoverDevices(ctx context.Context, urn URN, iface *net.Interface) ([]*Device, []error, error) {
	urls, errs, err := DiscoverURLs(ctx, urn, iface)
	if errors.Is(err, context.Canceled) {
		return nil, errs, err
	}

	var devices []*Device
	for _, manifestURL := range urls {
		// The discovery deadline is also the SSDP response collection window and
		// has normally elapsed by this point. Give each manifest fetch its own
		// bounded context while retaining values from the caller.
		manifestCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), discoveryTimeout)
		req, err := http.NewRequestWithContext(manifestCtx, http.MethodGet, manifestURL.String(), http.NoBody)
		if err != nil {
			cancel()
			errs = append(errs, fmt.Errorf("could not create manifest request for %v: %w", manifestURL, err))
			continue
		}
		rsp, err := http.DefaultClient.Do(req)
		if err != nil {
			cancel()
			errs = append(errs, fmt.Errorf("could not GET manifest %v: %w", manifestURL, err))
			continue
		}
		bytes, readErr := io.ReadAll(rsp.Body)
		closeErr := rsp.Body.Close()
		cancel()
		if readErr != nil {
			errs = append(errs, fmt.Errorf("could not read manifest %v: %w", manifestURL, readErr))
			continue
		}
		if closeErr != nil {
			errs = append(errs, fmt.Errorf("could not close manifest response %v: %w", manifestURL, closeErr))
			continue
		}
		if rsp.StatusCode < http.StatusOK || rsp.StatusCode >= http.StatusMultipleChoices {
			errs = append(errs, fmt.Errorf("could not GET manifest %v: HTTP status %s", manifestURL, rsp.Status))
			continue
		}

		manifest := ssdp.Document{}
		if err := xml.Unmarshal(bytes, &manifest); err != nil {
			errs = append(errs, err)
			continue
		}

		device, err := newDevice(manifestURL, manifest)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		devices = append(devices, device)
	}
	return devices, errs, err
}

// BroadcastDevice broadcasts the presence of a UPnP Device, with its SSDP/SCPD served via HTTP at addr.
func BroadcastDevice(ctx context.Context, d *Device, url string, iface *net.Interface, notifyInterval time.Duration) error {
	if notifyInterval <= 0 {
		return fmt.Errorf("notify interval must be positive")
	}
	conn, err := net.ListenMulticastUDP("udp", iface, ssdpBroadcastAddr)
	if err != nil {
		return fmt.Errorf("could not listen on %v: %v", ssdpBroadcastAddr, err)
	}
	p := ipv4.NewPacketConn(conn)
	_ = p.SetMulticastTTL(2)
	if iface != nil {
		_ = p.SetMulticastInterface(iface)
	}

	var once sync.Once
	closeConn := func() {
		once.Do(func() {
			_ = conn.Close()
		})
	}
	defer closeConn()

	go func() {
		<-ctx.Done()
		closeConn()
	}()

	log, _ := logger.FromContext(ctx)
	log.WithField("httpu.listener", ssdpBroadcastAddr).Info("serving HTTPU")
	s := &httpu.Server{
		Handler: func(r *http.Request) []httpu.Response {
			switch r.Method {
			case discoverMethod:
				return handleDiscover(r, d, url)
			case notifyMethod:
				// A device should not do anything with NOTIFY messages from other devices.
				return nil
			default:
				log.Warning("unknown method")
				return nil
			}
		},
	}

	sendAlive := func() {
		reqs := notifyAliveRequests(ctx, d, url)
		for _, req := range reqs {
			delay := time.Duration(rand.Int63n(int64(100 * time.Millisecond)))
			go func(req *http.Request) {
				if !s.Running() {
					return
				}

				timer := time.NewTimer(delay)
				defer timer.Stop()

				select {
				case <-timer.C:
				case <-ctx.Done():
					return
				}

				if !s.Running() {
					return
				}

				pkt := httpu.SerializeRequest(req)
				if _, err := conn.WriteTo(pkt, ssdpBroadcastAddr); err != nil {
					if err := httpu.Send(req, 1, iface); err != nil {
						log.Warning(err.Error())
					}
				}
				log := log.WithField("httpu.method", req.Method)
				log = log.WithField("httpu.notification.type", req.Header.Get("Nt"))
				log.Debug("sent ssdp:alive message")
			}(req)
		}
	}

	go func() {
		// UPnP 1.0 mandates immediate announcement on startup
		time.Sleep(50 * time.Millisecond)
		sendAlive()

		ticker := time.NewTicker(notifyInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sendAlive()
			}
		}
	}()

	err = s.Serve(conn)

	if ctx.Err() != nil {
		return nil
	}
	return err
}

func discoverRequest(ctx context.Context, urn URN) *http.Request {
	req, _ := http.NewRequestWithContext(ctx, discoverMethod, "", http.NoBody)
	req.URL = discoverURL
	req.Host = ssdpBroadcastAddr.String()
	req.Header = http.Header{
		"MAN": {`"ssdp:discover"`},
		"MX":  {"2"},
		"ST":  {string(urn)},
	}
	return req
}

func notifyAliveRequests(ctx context.Context, d *Device, url string) []*http.Request {
	var reqs []*http.Request
	bootID := d.BootID()

	makeReq := func(nt, usn string) *http.Request {
		req, _ := http.NewRequestWithContext(ctx, notifyMethod, "", http.NoBody)
		req.URL = discoverURL
		req.Host = ssdpBroadcastAddr.String()
		req.Header = http.Header{
			"Cache-Control":   {ssdpCacheControl},
			"Location":        {url},
			"Nt":              {nt},
			"Nts":             {notifyAlive},
			"Server":          {fmt.Sprintf("Linux/3.x UPnP/1.0 %s/1.0", d.ModelName)},
			"Usn":             {usn},
			"BootID.upnp.org": {fmt.Sprintf("%d", bootID)},
		}
		return req
	}

	reqs = append(reqs, makeReq(string(RootDevice), fmt.Sprintf("%s::%s", d.UDN, RootDevice)))
	reqs = append(reqs, makeReq(d.UDN, d.UDN))
	if d.DeviceType != "" {
		reqs = append(reqs, makeReq(string(d.DeviceType), fmt.Sprintf("%s::%s", d.UDN, d.DeviceType)))
	}
	for _, urn := range d.Services() {
		reqs = append(reqs, makeReq(string(urn), fmt.Sprintf("%s::%s", d.UDN, urn)))
	}
	return reqs
}

func notifyUpdateRequest(ctx context.Context, d *Device, url string) *http.Request {
	bootID := d.BootID()
	req, _ := http.NewRequestWithContext(ctx, notifyMethod, "", http.NoBody)
	req.URL = discoverURL
	req.Host = ssdpBroadcastAddr.String()
	req.Header = http.Header{
		"Location":            {url},
		"Nt":                  {string(RootDevice)},
		"Nts":                 {notifyUpdate},
		"Usn":                 {d.UDN},
		"BootID.upnp.org":     {fmt.Sprintf("%d", bootID)},
		"NextBootID.upnp.org": {fmt.Sprintf("%d", bootID+1)},
	}
	return req
}

func notifyByeByeRequests(ctx context.Context, d *Device) []*http.Request {
	var reqs []*http.Request
	bootID := d.BootID()

	makeReq := func(nt, usn string) *http.Request {
		req, _ := http.NewRequestWithContext(ctx, notifyMethod, "", http.NoBody)
		req.URL = discoverURL
		req.Host = ssdpBroadcastAddr.String()
		req.Header = http.Header{
			"Nt":              {nt},
			"Nts":             {notifyByeBye},
			"Usn":             {usn},
			"BootID.upnp.org": {fmt.Sprintf("%d", bootID)},
		}
		return req
	}

	reqs = append(reqs, makeReq(string(RootDevice), fmt.Sprintf("%s::%s", d.UDN, RootDevice)))
	reqs = append(reqs, makeReq(d.UDN, d.UDN))
	if d.DeviceType != "" {
		reqs = append(reqs, makeReq(string(d.DeviceType), fmt.Sprintf("%s::%s", d.UDN, d.DeviceType)))
	}
	for _, urn := range d.Services() {
		reqs = append(reqs, makeReq(string(urn), fmt.Sprintf("%s::%s", d.UDN, urn)))
	}
	return reqs
}

// SendUpdateNotification sends a ssdp:update notification, then increments the BootID.
func SendUpdateNotification(ctx context.Context, d *Device, url string, iface *net.Interface) error {
	req := notifyUpdateRequest(ctx, d, url)
	if err := httpu.Send(req, 1, iface); err != nil {
		return err
	}
	d.IncrementBootID()
	log, _ := logger.FromContext(ctx)
	log.WithField("httpu.method", req.Method).
		WithField("httpu.notification.type", req.Header.Get("Nts")).
		Debug("sent update notification")
	return nil
}

func NotifyByeBye(ctx context.Context, d *Device, url string, iface *net.Interface) {
	log, _ := logger.FromContext(ctx)

	reqs := notifyByeByeRequests(ctx, d)

	var wg sync.WaitGroup
	wg.Add(len(reqs))
	for _, req := range reqs {
		delay := time.Duration(rand.Int63n(int64(100 * time.Millisecond)))
		go func(req *http.Request) {
			defer wg.Done()
			<-time.After(delay)
			err := httpu.Send(req, 2, iface)
			if err != nil {
				log.Warning(err.Error())
			}
			log := log.WithField("httpu.method", req.Method)
			log = log.WithField("httpu.notification.type", req.Header.Get("Nt"))
			log.Debug("sent byebye message")
		}(req)
	}
	wg.Wait()
}

func handleDiscover(r *http.Request, d *Device, url string) []httpu.Response {
	log, _ := logger.FromContext(r.Context())

	man := r.Header.Get("Man")
	if !strings.Contains(strings.ToLower(man), "ssdp:discover") {
		log.Warning("request lacked correct MAN header")
		return nil
	}

	st := strings.TrimSpace(r.Header.Get("St"))
	bootID := fmt.Sprintf("%d", d.BootID())

	makeResp := func(target, usn string) httpu.Response {
		return httpu.Response{
			"CACHE-CONTROL":   ssdpCacheControl,
			"EXT":             "",
			"LOCATION":        url,
			"SERVER":          fmt.Sprintf("Linux/3.x UPnP/1.0 %s/1.0", d.ModelName),
			"ST":              target,
			"USN":             usn,
			"BOOTID.UPNP.ORG": bootID,
		}
	}

	allResponses := []httpu.Response{
		makeResp(string(RootDevice), fmt.Sprintf("%s::%s", d.UDN, RootDevice)),
		makeResp(d.UDN, d.UDN),
	}
	if d.DeviceType != "" {
		allResponses = append(allResponses, makeResp(string(d.DeviceType), fmt.Sprintf("%s::%s", d.UDN, d.DeviceType)))
	}
	for _, urn := range d.Services() {
		allResponses = append(allResponses, makeResp(string(urn), fmt.Sprintf("%s::%s", d.UDN, urn)))
	}

	if strings.EqualFold(st, string(All)) {
		return allResponses
	}

	for _, resp := range allResponses {
		if strings.EqualFold(resp["ST"], st) {
			return []httpu.Response{resp}
		}
	}

	return nil
}
