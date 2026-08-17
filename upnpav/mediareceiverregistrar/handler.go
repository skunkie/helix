// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package mediareceiverregistrar

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/ethulhu/helix/soap"
	"github.com/ethulhu/helix/upnpav"
)

type SOAPHandler struct {
	Interface
}

func (h SOAPHandler) Call(ctx context.Context, namespace, action string, in []byte) ([]byte, error) {
	if !strings.EqualFold(namespace, string(Version1)) {
		return nil, fmt.Errorf("invalid namespace: %q", namespace)
	}

	switch action {
	case isAuthorized:
		return h.isAuthorized(ctx, in)
	case registerDevice:
		return h.registerDevice(ctx, in)
	case isValid:
		return h.isValid(ctx, in)
	default:
		return nil, upnpav.ErrInvalidAction
	}
}

func (h SOAPHandler) isAuthorized(ctx context.Context, in []byte) ([]byte, error) {
	req := isAuthorizedRequest{}
	if err := xml.Unmarshal(in, &req); err != nil {
		return nil, upnpav.ErrInvalidArgs
	}

	result, err := h.Interface.IsAuthorized(ctx, req.DeviceID)
	if err != nil {
		return nil, err
	}

	rsp := isAuthorizedResponse{
		Result: result,
	}

	prefix := soap.DetectPrefix(in, isAuthorized)
	if prefix != "" {
		rsp.XMLName = xml.Name{Local: prefix + ":IsAuthorizedResponse"}
		rsp.Xmlns = []xml.Attr{{Name: xml.Name{Local: "xmlns:" + prefix}, Value: string(Version1)}}
	} else {
		rsp.XMLName = xml.Name{Space: string(Version1), Local: "IsAuthorizedResponse"}
	}

	return xml.Marshal(rsp)
}

func (h SOAPHandler) registerDevice(ctx context.Context, in []byte) ([]byte, error) {
	req := registerDeviceRequest{}
	if err := xml.Unmarshal(in, &req); err != nil {
		return nil, upnpav.ErrInvalidArgs
	}

	respMsg, err := h.Interface.RegisterDevice(ctx, req.RegistrationReqMsg)
	if err != nil {
		return nil, err
	}

	rsp := registerDeviceResponse{
		RegistrationRespMsg: respMsg,
	}

	prefix := soap.DetectPrefix(in, registerDevice)
	if prefix != "" {
		rsp.XMLName = xml.Name{Local: prefix + ":RegisterDeviceResponse"}
		rsp.Xmlns = []xml.Attr{{Name: xml.Name{Local: "xmlns:" + prefix}, Value: string(Version1)}}
	} else {
		rsp.XMLName = xml.Name{Space: string(Version1), Local: "RegisterDeviceResponse"}
	}

	return xml.Marshal(rsp)
}

func (h SOAPHandler) isValid(ctx context.Context, in []byte) ([]byte, error) {
	req := isValidRequest{}
	if err := xml.Unmarshal(in, &req); err != nil {
		return nil, upnpav.ErrInvalidArgs
	}

	result, err := h.Interface.IsValid(ctx, req.DeviceID)
	if err != nil {
		return nil, err
	}

	rsp := isValidResponse{
		Result: result,
	}

	prefix := soap.DetectPrefix(in, isValid)
	if prefix != "" {
		rsp.XMLName = xml.Name{Local: prefix + ":IsValidResponse"}
		rsp.Xmlns = []xml.Attr{{Name: xml.Name{Local: "xmlns:" + prefix}, Value: string(Version1)}}
	} else {
		rsp.XMLName = xml.Name{Space: string(Version1), Local: "IsValidResponse"}
	}

	return xml.Marshal(rsp)
}
