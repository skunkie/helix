// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package mediareceiverregistrar

import (
	"encoding/xml"
)

const (
	isAuthorized   = "IsAuthorized"
	isValid        = "IsValid"
	registerDevice = "RegisterDevice"
)

type (
	isAuthorizedRequest struct {
		XMLName  xml.Name `xml:"urn:microsoft.com:service:X_MS_MediaReceiverRegistrar:1 IsAuthorized"`
		DeviceID string   `xml:"DeviceID" scpd:"A_ARG_TYPE_DeviceID,string"`
	}
	isAuthorizedResponse struct {
		XMLName xml.Name
		Xmlns   []xml.Attr `xml:",attr,omitempty"`
		Result  int        `xml:"Result" scpd:"A_ARG_TYPE_Result,int"`
	}

	registerDeviceRequest struct {
		XMLName            xml.Name `xml:"urn:microsoft.com:service:X_MS_MediaReceiverRegistrar:1 RegisterDevice"`
		RegistrationReqMsg []byte   `xml:"RegistrationReqMsg" scpd:"A_ARG_TYPE_RegistrationReqMsg,bin.base64"`
	}
	registerDeviceResponse struct {
		XMLName             xml.Name
		Xmlns               []xml.Attr `xml:",attr,omitempty"`
		RegistrationRespMsg []byte     `xml:"RegistrationRespMsg" scpd:"A_ARG_TYPE_RegistrationRespMsg,bin.base64"`
	}

	isValidRequest struct {
		XMLName  xml.Name `xml:"urn:microsoft.com:service:X_MS_MediaReceiverRegistrar:1 IsValid"`
		DeviceID string   `xml:"DeviceID" scpd:"A_ARG_TYPE_DeviceID,string"`
	}
	isValidResponse struct {
		XMLName xml.Name
		Xmlns   []xml.Attr `xml:",attr,omitempty"`
		Result  int        `xml:"Result" scpd:"A_ARG_TYPE_Result,int"`
	}
)
