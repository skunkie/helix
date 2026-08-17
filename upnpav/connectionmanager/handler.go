// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package connectionmanager

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/ethulhu/helix/soap"
	"github.com/ethulhu/helix/upnpav"
	"github.com/ethulhu/helix/xmltypes"
)

type SOAPHandler struct {
	Interface
}

func (h SOAPHandler) Call(ctx context.Context, namespace, action string, in []byte) ([]byte, error) {
	if !strings.EqualFold(namespace, string(Version1)) {
		return nil, fmt.Errorf("invalid namespace: %q", namespace)
	}

	switch action {
	case getProtocolInfo:
		return h.getProtocolInfo(ctx, in)
	case getCurrentConnectionIDs:
		return h.getCurrentConnectionIDs(ctx, in)
	case getCurrentConnectionInfo:
		return h.getCurrentConnectionInfo(ctx, in)
	default:
		return nil, upnpav.ErrInvalidAction
	}
}

func (h SOAPHandler) getProtocolInfo(ctx context.Context, in []byte) ([]byte, error) {
	req := getProtocolInfoRequest{}
	if err := xml.Unmarshal(in, &req); err != nil {
		return nil, upnpav.ErrInvalidArgs
	}

	sources, sinks, err := h.Interface.ProtocolInfo(ctx)
	if err != nil {
		return nil, err
	}

	rsp := getProtocolInfoResponse{
		Sources: commaSeparatedProtocolInfos(sources),
		Sinks:   commaSeparatedProtocolInfos(sinks),
	}

	prefix := soap.DetectPrefix(in, getProtocolInfo)
	if prefix != "" {
		rsp.XMLName = xml.Name{Local: prefix + ":GetProtocolInfoResponse"}
		rsp.Xmlns = []xml.Attr{{Name: xml.Name{Local: "xmlns:" + prefix}, Value: string(Version1)}}
	} else {
		rsp.XMLName = xml.Name{Space: string(Version1), Local: "GetProtocolInfoResponse"}
	}

	return xml.Marshal(rsp)
}

func (h SOAPHandler) getCurrentConnectionIDs(ctx context.Context, in []byte) ([]byte, error) {
	req := getCurrentConnectionIDsRequest{}
	if err := xml.Unmarshal(in, &req); err != nil {
		return nil, upnpav.ErrInvalidArgs
	}

	ids, err := h.Interface.CurrentConnectionIDs(ctx)
	if err != nil {
		return nil, err
	}

	rsp := getCurrentConnectionIDsResponse{
		ConnectionIDs: xmltypes.CommaSeparatedInts(ids),
	}

	prefix := soap.DetectPrefix(in, getCurrentConnectionIDs)
	if prefix != "" {
		rsp.XMLName = xml.Name{Local: prefix + ":GetCurrentConnectionIDsResponse"}
		rsp.Xmlns = []xml.Attr{{Name: xml.Name{Local: "xmlns:" + prefix}, Value: string(Version1)}}
	} else {
		rsp.XMLName = xml.Name{Space: string(Version1), Local: "GetCurrentConnectionIDsResponse"}
	}

	return xml.Marshal(rsp)
}

func (h SOAPHandler) getCurrentConnectionInfo(ctx context.Context, in []byte) ([]byte, error) {
	req := getCurrentConnectionInfoRequest{}
	if err := xml.Unmarshal(in, &req); err != nil {
		return nil, upnpav.ErrInvalidArgs
	}

	info, err := h.Interface.CurrentConnectionInfo(ctx, req.ConnectionID)
	if err != nil {
		return nil, err
	}

	rsp := getCurrentConnectionInfoResponse{
		RcsID:                 info.RcsID,
		AVTransportID:         info.AVTransportID,
		ProtocolInfo:          info.ProtocolInfo,
		PeerConnectionManager: info.PeerConnectionManager,
		PeerConnectionID:      info.PeerConnectionID,
		Direction:             direction(info.Direction),
		Status:                status(info.Status),
	}

	prefix := soap.DetectPrefix(in, getCurrentConnectionInfo)
	if prefix != "" {
		rsp.XMLName = xml.Name{Local: prefix + ":GetCurrentConnectionInfoResponse"}
		rsp.Xmlns = []xml.Attr{{Name: xml.Name{Local: "xmlns:" + prefix}, Value: string(Version1)}}
	} else {
		rsp.XMLName = xml.Name{Space: string(Version1), Local: "GetCurrentConnectionInfoResponse"}
	}

	return xml.Marshal(rsp)
}
