// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package connectionmanager

import (
	"context"

	"github.com/ethulhu/helix/upnpav"
)

type Server struct {
	sources []upnpav.ProtocolInfo
	sinks   []upnpav.ProtocolInfo
}

// NewServer creates a new default ConnectionManager server.
// If sources is nil, upnpav.DefaultProtocolInfo is used.
func NewServer(sources, sinks []upnpav.ProtocolInfo) *Server {
	if sources == nil {
		var defaultSources commaSeparatedProtocolInfos
		_ = defaultSources.UnmarshalText([]byte(upnpav.DefaultProtocolInfo))
		sources = defaultSources
	}
	return &Server{
		sources: sources,
		sinks:   sinks,
	}
}

func (s *Server) ProtocolInfo(_ context.Context) ([]upnpav.ProtocolInfo, []upnpav.ProtocolInfo, error) {
	return s.sources, s.sinks, nil
}

func (s *Server) CurrentConnectionIDs(_ context.Context) ([]int, error) {
	return []int{0}, nil
}

func (s *Server) CurrentConnectionInfo(_ context.Context, connectionID int) (*ConnectionInfo, error) {
	if connectionID != 0 {
		return nil, ErrInvalidConnectionReference
	}
	return &ConnectionInfo{
		RcsID:                 -1,
		AVTransportID:         -1,
		ProtocolInfo:          "http-get:*:*:*",
		PeerConnectionManager: "/",
		PeerConnectionID:      -1,
		Direction:             Output,
		Status:                StatusOK,
	}, nil
}
