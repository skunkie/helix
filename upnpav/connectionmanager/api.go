// SPDX-FileCopyrightText: 2020 Ethel Morgan
// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package connectionmanager

import (
	"context"

	"github.com/ethulhu/helix/upnp"
	"github.com/ethulhu/helix/upnp/scpd"
	"github.com/ethulhu/helix/upnpav"
	"github.com/ethulhu/helix/xmltypes"
)

type (
	Direction string
	Status    string

	ConnectionInfo struct {
		RcsID                 int
		AVTransportID         int
		ProtocolInfo          string
		PeerConnectionManager string
		PeerConnectionID      int
		Direction             Direction
		Status                Status
	}

	Interface interface {
		// ProtocolInfo lists the protocols that the device can send and receive, respectively.
		ProtocolInfo(ctx context.Context) (sources []upnpav.ProtocolInfo, sinks []upnpav.ProtocolInfo, err error)

		// CurrentConnectionIDs returns the list of current connection IDs.
		CurrentConnectionIDs(ctx context.Context) ([]int, error)

		// CurrentConnectionInfo returns info about a specific connection.
		CurrentConnectionInfo(ctx context.Context, connectionID int) (*ConnectionInfo, error)
	}
)

const (
	Input  = Direction("Input")
	Output = Direction("Output")
)

const (
	StatusOK                    = Status("OK")
	StatusContentFormatMismatch = Status("ContentFormatMismatch")
	StatusInsufficientBandwidth = Status("InsufficientBandwidth")
	StatusUnreliableChannel     = Status("UnreliableChannel")
	StatusUnknown               = Status("Unknown")
)

const (
	Version1  = upnp.URN("urn:schemas-upnp-org:service:ConnectionManager:1")
	Version2  = upnp.URN("urn:schemas-upnp-org:service:ConnectionManager:2")
	ServiceID = upnp.ServiceID("urn:upnp-org:serviceId:ConnectionManager")
)

var (
	ErrInvalidConnectionReference = upnpav.Error{Code: 706, Description: "Invalid connection reference"}
)

var SCPD = scpd.Must(scpd.Merge(
	scpd.Must(scpd.FromAction(getProtocolInfo, getProtocolInfoRequest{}, getProtocolInfoResponse{
		Sources: func() commaSeparatedProtocolInfos {
			var infos commaSeparatedProtocolInfos
			_ = infos.UnmarshalText([]byte(upnpav.DefaultProtocolInfo))
			return infos
		}(),
		Sinks: nil,
	})),
	scpd.Must(scpd.FromAction(getCurrentConnectionIDs, getCurrentConnectionIDsRequest{}, getCurrentConnectionIDsResponse{
		ConnectionIDs: xmltypes.CommaSeparatedInts{0},
	})),
	scpd.Must(scpd.FromAction(getCurrentConnectionInfo, getCurrentConnectionInfoRequest{ConnectionID: 0}, getCurrentConnectionInfoResponse{
		RcsID:                 -1,
		AVTransportID:         -1,
		ProtocolInfo:          "http-get:*:*:*",
		PeerConnectionManager: "/",
		PeerConnectionID:      -1,
		Direction:             output,
		Status:                ok,
	})),
))
