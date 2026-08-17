// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package mediareceiverregistrar

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
		panic(fmt.Sprintf("could not marshal MediaReceiverRegistrar SOAP request: %v", err))
	}

	rsp, err := c.Call(ctx, string(Version1), method, req)
	if err != nil {
		return upnpav.MaybeError(err)
	}
	return xml.Unmarshal(rsp, output)
}

func (c *client) IsAuthorized(ctx context.Context, deviceID string) (int, error) {
	req := isAuthorizedRequest{DeviceID: deviceID}
	rsp := isAuthorizedResponse{}
	if err := c.call(ctx, isAuthorized, req, &rsp); err != nil {
		return 0, err
	}
	return rsp.Result, nil
}

func (c *client) RegisterDevice(ctx context.Context, reqData []byte) ([]byte, error) {
	req := registerDeviceRequest{RegistrationReqMsg: reqData}
	rsp := registerDeviceResponse{}
	if err := c.call(ctx, registerDevice, req, &rsp); err != nil {
		return nil, err
	}
	return rsp.RegistrationRespMsg, nil
}

func (c *client) IsValid(ctx context.Context, deviceID string) (int, error) {
	req := isValidRequest{DeviceID: deviceID}
	rsp := isValidResponse{}
	if err := c.call(ctx, isValid, req, &rsp); err != nil {
		return 0, err
	}
	return rsp.Result, nil
}
