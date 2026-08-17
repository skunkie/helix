// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package mediareceiverregistrar

import (
	"context"

	"github.com/ethulhu/helix/upnp"
	"github.com/ethulhu/helix/upnp/scpd"
)

type (
	Interface interface {
		// IsAuthorized returns whether the given device is authorized (1 for yes, 0 for no).
		IsAuthorized(ctx context.Context, deviceID string) (int, error)

		// RegisterDevice registers a device.
		RegisterDevice(ctx context.Context, reqData []byte) ([]byte, error)

		// IsValid returns whether the given device registration is valid (1 for yes, 0 for no).
		IsValid(ctx context.Context, deviceID string) (int, error)
	}
)

const (
	Version1  = upnp.URN("urn:microsoft.com:service:X_MS_MediaReceiverRegistrar:1")
	ServiceID = upnp.ServiceID("urn:microsoft.com:serviceId:X_MS_MediaReceiverRegistrar")
)

var SCPD = scpd.Must(scpd.Merge(
	scpd.Must(scpd.FromAction(isAuthorized, isAuthorizedRequest{DeviceID: "uuid:0000"}, isAuthorizedResponse{Result: 1})),
	scpd.Must(scpd.FromAction(registerDevice, registerDeviceRequest{RegistrationReqMsg: []byte{}}, registerDeviceResponse{RegistrationRespMsg: []byte{}})),
	scpd.Must(scpd.FromAction(isValid, isValidRequest{DeviceID: "uuid:0000"}, isValidResponse{Result: 1})),
))
