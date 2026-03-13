// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

package upnp

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/ethulhu/helix/logger"
)

type (
	// DeviceCache is an automatically refreshing cache of UPnP devices, addressable by UDN.
	DeviceCache struct {
		urn   URN
		iface *net.Interface

		mu      sync.Mutex
		devices map[string]*Device
	}

	DeviceCacheOptions struct {
		InitialRefresh time.Duration
		StableRefresh  time.Duration

		Interface *net.Interface
	}
)

const (
	discoveryTimeout = 2 * time.Second
)

// NewDeviceCache returns a DeviceCache searching for the given URN, every refresh period, optionally on a specific network interface.
func NewDeviceCache(ctx context.Context, urn URN, options DeviceCacheOptions) *DeviceCache {
	d := &DeviceCache{
		urn:   urn,
		iface: options.Interface,

		devices: map[string]*Device{},
	}

	d.Refresh(ctx)

	go func() {
		var ticker *time.Ticker
		if len(d.Devices()) > 0 {
			ticker = time.NewTicker(options.StableRefresh)
		} else {
			ticker = time.NewTicker(options.InitialRefresh)
		}
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				hadDevices := len(d.Devices()) > 0
				d.Refresh(ctx)
				hasDevices := len(d.Devices()) > 0

				switch {
				case !hadDevices && hasDevices:
					ticker.Reset(options.StableRefresh)
				case hadDevices && !hasDevices:
					ticker.Reset(options.InitialRefresh)
				}
			}
		}
	}()

	return d
}

// Refresh forces the DeviceCache to update itself by discovering UPnP devices.
func (d *DeviceCache) Refresh(ctx context.Context) {
	d.mu.Lock()
	defer d.mu.Unlock()

	log, ctx := logger.FromContext(ctx)
	log.AddField("upnp.urn", d.urn)

	ctx, cancel := context.WithTimeout(ctx, discoveryTimeout)
	defer cancel()

	devices, _, err := DiscoverDevices(ctx, d.urn, d.iface)
	if err != nil {
		log.Warning("could not find UPnP devices")
		return
	}

	newDevices := map[string]*Device{}
	for _, device := range devices {
		newDevices[device.UDN] = device
	}

	d.devices = newDevices
	log.Debug("updated UPnP device cache")
}

// Devices lists all currently known Devices.
func (d *DeviceCache) Devices() []*Device {
	d.mu.Lock()
	defer d.mu.Unlock()

	var devices []*Device
	for _, device := range d.devices {
		devices = append(devices, device)
	}
	return devices
}

// DeviceByUDN returns the Device with a given UDN.
func (d *DeviceCache) DeviceByUDN(udn string) (*Device, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	device, ok := d.devices[udn]
	return device, ok
}
