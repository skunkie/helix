// SPDX-FileCopyrightText: 2020 Ethel Morgan
// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package ssdp

import "encoding/xml"

const xmlns = "urn:schemas-upnp-org:device-1-0"

var Version = SpecVersion{
	Major: 1,
	Minor: 0,
}

type (
	Document struct {
		XMLName     xml.Name    `xml:"urn:schemas-upnp-org:device-1-0 root"`
		NSDLNA      string      `xml:"xmlns:dlna,attr"`
		NSSEC       string      `xml:"xmlns:sec,attr"`
		SpecVersion SpecVersion `xml:"specVersion"`
		Device      Device      `xml:"device"`
	}

	SpecVersion struct {
		Major int `xml:"major"`
		Minor int `xml:"minor"`
	}

	Device struct {
		DeviceType       string `xml:"deviceType,omitempty"`
		FriendlyName     string `xml:"friendlyName,omitempty"`
		Manufacturer     string `xml:"manufacturer,omitempty"`
		ManufacturerURL  string `xml:"manufacturerURL,omitempty"`
		ModelDescription string `xml:"modelDescription,omitempty"`
		ModelName        string `xml:"modelName,omitempty"`
		ModelNumber      string `xml:"modelNumber,omitempty"`
		ModelURL         string `xml:"modelURL,omitempty"`
		SerialNumber     string `xml:"serialNumber,omitempty"`
		UDN              string `xml:"UDN,omitempty"`

		DLNACAP *DLNACAP `xml:"dlna:X_DLNACAP,omitempty"`
		DLNADOC []string `xml:"dlna:X_DLNADOC,omitempty"`
		SecCap  string   `xml:"sec:ProductCap,omitempty"`
		XSecCap string   `xml:"sec:X_ProductCap,omitempty"`

		DeviceList      *DeviceList `xml:"deviceList,omitempty"`
		IconList        *IconList   `xml:"iconList,omitempty"`
		ServiceList     ServiceList `xml:"serviceList"`
		PresentationURL string      `xml:"presentationURL,omitempty"`
	}
	DLNACAP    struct{}
	DeviceList struct {
		Devices []Device `xml:"device"`
	}
	IconList struct {
		Icons []Icon `xml:"icon"`
	}
	ServiceList struct {
		Services []Service `xml:"service"`
	}
	Icon struct {
		MIMEType string `xml:"mimetype"`
		Width    int    `xml:"width"`
		Height   int    `xml:"height"`
		Depth    int    `xml:"depth"`
		URL      string `xml:"url"`
	}
	Service struct {
		ServiceType string `xml:"serviceType"` // URN
		ServiceID   string `xml:"serviceId"`
		ControlURL  string `xml:"controlURL"`
		EventSubURL string `xml:"eventSubURL"`
		SCPDURL     string `xml:"SCPDURL"`
	}
)
