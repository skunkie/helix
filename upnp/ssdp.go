// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

package upnp

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/ethulhu/helix/logger"
	"github.com/ethulhu/helix/upnp/httpu"
	"github.com/ethulhu/helix/upnp/ssdp"
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

	var devices []*Device
	for _, manifestURL := range urls {
		rsp, err := http.Get(manifestURL.String())
		if err != nil {
			errs = append(errs, fmt.Errorf("could not GET manifest %v: %w", manifestURL, err))
			continue
		}
		bytes, _ := io.ReadAll(rsp.Body)

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
	conn, err := net.ListenMulticastUDP("udp", iface, ssdpBroadcastAddr)
	if err != nil {
		return fmt.Errorf("could not listen on %v: %v", ssdpBroadcastAddr, err)
	}

	var once sync.Once
	closeConn := func() {
		once.Do(func() {
			conn.Close()
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

	go func() {
		ticker := time.NewTicker(notifyInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				var reqs []*http.Request
				for _, urn := range d.allURNs() {
					reqs = append(reqs, notifyAliveRequest(ctx, d, urn, url))
				}
				reqs = append(reqs, notifyUpdateRequest(ctx, d, url))

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

						if err := httpu.Send(req, 1, iface); err != nil {
							log.Warning(err.Error())
						}
						log := log.WithField("httpu.method", req.Method)
						log = log.WithField("httpu.notification.type", req.Header.Get("Nt"))
						log.Debug("sent alive message")
					}(req)
				}
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

func notifyAliveRequest(ctx context.Context, d *Device, urn URN, url string) *http.Request {
	req, _ := http.NewRequestWithContext(ctx, notifyMethod, "", http.NoBody)
	req.URL = discoverURL
	req.Host = ssdpBroadcastAddr.String()
	req.Header = http.Header{
		"Cache-Control":   {ssdpCacheControl},
		"Location":        {url},
		"Nt":              {string(urn)},
		"Nts":             {notifyAlive},
		"Server":          {fmt.Sprintf("%s %s", d.ModelName, d.ModelNumber)},
		"Usn":             {fmt.Sprintf("%s::%s", d.UDN, urn)},
		"BootID.upnp.org": {fmt.Sprintf("%d", d.BootID())},
	}
	return req
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
	for _, urn := range d.allURNs() {
		req, _ := http.NewRequestWithContext(ctx, notifyMethod, "", http.NoBody)
		req.URL = discoverURL
		req.Host = ssdpBroadcastAddr.String()
		req.Header = http.Header{
			"Nt":              {string(urn)},
			"Nts":             {notifyByeBye},
			"Usn":             {fmt.Sprintf("%s::%s", d.UDN, urn)},
			"BootID.upnp.org": {fmt.Sprintf("%d", bootID)},
		}
		reqs = append(reqs, req)
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
		Info("sent update notification")
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

	if r.Header.Get("Man") != `"ssdp:discover"` {
		log.Warning("request lacked correct MAN header")
		return nil
	}

	st := URN(r.Header.Get("St"))

	ok := false
	for _, urn := range d.allURNs() {
		ok = ok || urn == st
	}
	if st == All || ok {
		bootID := fmt.Sprintf("%d", d.BootID())
		responses := []httpu.Response{{
			"CACHE-CONTROL":   ssdpCacheControl,
			"EXT":             "",
			"LOCATION":        url,
			"SERVER":          fmt.Sprintf("%s %s", d.ModelName, d.ModelNumber),
			"ST":              d.UDN,
			"USN":             d.UDN,
			"BOOTID.UPNP.ORG": bootID,
		}}
		for _, urn := range d.allURNs() {
			responses = append(responses, httpu.Response{
				"CACHE-CONTROL":   ssdpCacheControl,
				"EXT":             "",
				"LOCATION":        url,
				"SERVER":          fmt.Sprintf("%s %s", d.ModelName, d.ModelNumber),
				"ST":              string(urn),
				"USN":             fmt.Sprintf("%s::%s", d.UDN, urn),
				"BOOTID.UPNP.ORG": bootID,
			})
		}
		return responses
	}
	return nil
}
