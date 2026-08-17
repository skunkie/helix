// SPDX-FileCopyrightText: 2020 Ethel Morgan
// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package connectionmanager

import (
	"context"
	"encoding/xml"
	"fmt"

	"github.com/ethulhu/helix/soap"
	"github.com/ethulhu/helix/upnpav"
)

type (
	client struct{ soap.Interface }
)

func NewClient(soapClient soap.Interface) Interface {
	return &client{soapClient}
}

func (c *client) call(ctx context.Context, method string, input, output interface{}) error {
	req, err := xml.Marshal(input)
	if err != nil {
		panic(fmt.Sprintf("could not marshal ConnectionManager SOAP request: %v", err))
	}

	rsp, err := c.Call(ctx, string(Version1), method, req)
	if err != nil {
		return upnpav.MaybeError(err)
	}
	return xml.Unmarshal(rsp, output)
}

func (c *client) ProtocolInfo(ctx context.Context) ([]upnpav.ProtocolInfo, []upnpav.ProtocolInfo, error) {
	req := getProtocolInfoRequest{}
	rsp := getProtocolInfoResponse{}
	if err := c.call(ctx, getProtocolInfo, req, &rsp); err != nil {
		return nil, nil, err
	}
	return rsp.Sources, rsp.Sinks, nil
}

func (c *client) CurrentConnectionIDs(ctx context.Context) ([]int, error) {
	req := getCurrentConnectionIDsRequest{}
	rsp := getCurrentConnectionIDsResponse{}
	if err := c.call(ctx, getCurrentConnectionIDs, req, &rsp); err != nil {
		return nil, err
	}
	return []int(rsp.ConnectionIDs), nil
}

func (c *client) CurrentConnectionInfo(ctx context.Context, connectionID int) (*ConnectionInfo, error) {
	req := getCurrentConnectionInfoRequest{ConnectionID: connectionID}
	rsp := getCurrentConnectionInfoResponse{}
	if err := c.call(ctx, getCurrentConnectionInfo, req, &rsp); err != nil {
		return nil, err
	}
	return &ConnectionInfo{
		RcsID:                 rsp.RcsID,
		AVTransportID:         rsp.AVTransportID,
		ProtocolInfo:          rsp.ProtocolInfo,
		PeerConnectionManager: rsp.PeerConnectionManager,
		PeerConnectionID:      rsp.PeerConnectionID,
		Direction:             Direction(rsp.Direction),
		Status:                Status(rsp.Status),
	}, nil
}
