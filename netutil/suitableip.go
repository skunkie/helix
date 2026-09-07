// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

package netutil

import (
	"errors"
	"fmt"
	"net"
)

func SuitableIP(iface *net.Interface) (net.IP, error) {
	addrs, err := net.InterfaceAddrs()
	if iface != nil {
		addrs, err = iface.Addrs()
	}
	if err != nil {
		return nil, fmt.Errorf("could not list addresses: %w", err)
	}
	return suitableIP(addrs, iface == nil)
}

func suitableIP(addrs []net.Addr, requirePrivate bool) (net.IP, error) {
	err := errors.New("interface has no addresses")
	for _, addr := range addrs {
		addr, ok := addr.(*net.IPNet)
		if !ok {
			err = errors.New("interface has no IP addresses")
			continue
		}
		if addr.IP.To4() == nil {
			err = errors.New("interface has no IPv4 addresses")
			continue
		}
		ip := addr.IP.To4()

		// Default IP must be a "LAN IP".
		if requirePrivate && !ip.IsPrivate() {
			err = errors.New("interface has no Private IPv4 addresses")
			continue
		}

		return ip, nil
	}
	return nil, err
}
