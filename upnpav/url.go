// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package upnpav

import (
	"net/url"
)

type (
	// URL wraps url.URL to provide text XML marshaling.
	URL struct {
		url.URL
	}
)

func ParseURL(raw string) (*URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	return &URL{*parsed}, nil
}

func (u URL) MarshalText() ([]byte, error) {
	return []byte(u.String()), nil
}

func (u *URL) UnmarshalText(raw []byte) error {
	parsed, err := url.Parse(string(raw))
	if err != nil {
		return err
	}
	*u = URL{*parsed}
	return nil
}
