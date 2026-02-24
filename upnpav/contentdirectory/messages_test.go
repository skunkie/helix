// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

package contentdirectory

import (
	"encoding/xml"
	"reflect"
	"testing"

	"github.com/ethulhu/helix/upnp/scpd"
	"github.com/ethulhu/helix/xmltypes"
)

func TestSCPD(t *testing.T) {
	actions := []struct {
		name     string
		req, rsp interface{}
	}{
		{getSearchCapabilities, getSearchCapabilitiesRequest{}, getSearchCapabilitiesResponse{}},
		{getSortCapabilities, getSortCapabilitiesRequest{}, getSortCapabilitiesResponse{}},
		{getSystemUpdateID, getSystemUpdateIDRequest{}, getSystemUpdateIDResponse{}},
		{xGetFeatureList, xGetFeatureListRequest{}, xGetFeatureListResponse{}},

		{browse, browseRequest{}, browseResponse{}},
		{searchA, searchRequest{}, searchResponse{}},

		{createObject, createObjectRequest{}, createObjectResponse{}},
		{destroyObject, destroyObjectRequest{}, destroyObjectResponse{}},
		{updateObject, updateObjectRequest{}, updateObjectResponse{}},

		{deleteResource, deleteResourceRequest{}, deleteResourceResponse{}},
		{exportResource, exportResourceRequest{}, exportResourceResponse{}},
		{importResource, importResourceRequest{}, importResourceResponse{}},
		{stopTransferResource, stopTransferResourceRequest{}, stopTransferResourceResponse{}},
		{getTransferProgress, getTransferProgressRequest{}, getTransferProgressResponse{}},

		{createReference, createReferenceRequest{}, createReferenceResponse{}},
	}

	var docs []scpd.Document
	for _, action := range actions {
		doc, err := scpd.FromAction(action.name, action.req, action.rsp)
		if err != nil {
			t.Errorf("SCPD definition for action %q is broken: %v", action.name, err)
		}
		docs = append(docs, doc)
	}

	if _, err := scpd.Merge(docs...); err != nil {
		t.Errorf("could not merge SCPDs: %v", err)
	}
}

func TestGetSearchCapabilities(t *testing.T) {
	t.Run("request", func(t *testing.T) {
		input := `<u:GetSearchCapabilities xmlns:u="urn:schemas-upnp-org:service:ContentDirectory:1"></u:GetSearchCapabilities>`
		expected := getSearchCapabilitiesRequest{
			XMLName: xml.Name{Space: "urn:schemas-upnp-org:service:ContentDirectory:1", Local: "GetSearchCapabilities"},
		}

		var actual getSearchCapabilitiesRequest
		if err := xml.Unmarshal([]byte(input), &actual); err != nil {
			t.Fatalf("could not unmarshal: %v", err)
		}

		if !reflect.DeepEqual(expected, actual) {
			t.Fatalf("unexpected value, got: %v, want: %v", actual, expected)
		}
	})
	t.Run("response", func(t *testing.T) {
		input := getSearchCapabilitiesResponse{
			XMLName:      xml.Name{Local: "u:GetSearchCapabilitiesResponse"},
			Xmlns:        []xml.Attr{{Name: xml.Name{Local: "xmlns:u"}, Value: string(Version1)}},
			Capabilities: xmltypes.CommaSeparatedStrings([]string{"foo", "bar"}),
		}
		expected := `<u:GetSearchCapabilitiesResponse xmlns:u="urn:schemas-upnp-org:service:ContentDirectory:1"><SearchCaps>foo,bar</SearchCaps></u:GetSearchCapabilitiesResponse>`

		actual, err := xml.Marshal(input)
		if err != nil {
			t.Fatalf("could not marshal: %v", err)
		}

		if string(actual) != expected {
			t.Fatalf("unexpected value, got: %v, want: %v", string(actual), expected)
		}
	})
}

func TestGetSortCapabilities(t *testing.T) {
	t.Run("request", func(t *testing.T) {
		input := `<u:GetSortCapabilities xmlns:u="urn:schemas-upnp-org:service:ContentDirectory:1"></u:GetSortCapabilities>`
		expected := getSortCapabilitiesRequest{
			XMLName: xml.Name{Space: "urn:schemas-upnp-org:service:ContentDirectory:1", Local: "GetSortCapabilities"},
		}

		var actual getSortCapabilitiesRequest
		if err := xml.Unmarshal([]byte(input), &actual); err != nil {
			t.Fatalf("could not unmarshal: %v", err)
		}

		if !reflect.DeepEqual(expected, actual) {
			t.Fatalf("unexpected value, got: %v, want: %v", actual, expected)
		}
	})
	t.Run("response", func(t *testing.T) {
		input := getSortCapabilitiesResponse{
			XMLName:      xml.Name{Local: "u:GetSortCapabilitiesResponse"},
			Xmlns:        []xml.Attr{{Name: xml.Name{Local: "xmlns:u"}, Value: string(Version1)}},
			Capabilities: xmltypes.CommaSeparatedStrings([]string{"foo", "bar"}),
		}
		expected := `<u:GetSortCapabilitiesResponse xmlns:u="urn:schemas-upnp-org:service:ContentDirectory:1"><SortCaps>foo,bar</SortCaps></u:GetSortCapabilitiesResponse>`

		actual, err := xml.Marshal(input)
		if err != nil {
			t.Fatalf("could not marshal: %v", err)
		}

		if string(actual) != expected {
			t.Fatalf("unexpected value, got: %v, want: %v", string(actual), expected)
		}
	})
}

func TestXGetFeatureList(t *testing.T) {
	t.Run("request", func(t *testing.T) {
		input := `<u:X_GetFeatureList xmlns:u="urn:schemas-upnp-org:service:ContentDirectory:1"></u:X_GetFeatureList>`
		expected := xGetFeatureListRequest{
			XMLName: xml.Name{Space: "urn:schemas-upnp-org:service:ContentDirectory:1", Local: "X_GetFeatureList"},
		}

		var actual xGetFeatureListRequest
		if err := xml.Unmarshal([]byte(input), &actual); err != nil {
			t.Fatalf("could not unmarshal: %v", err)
		}

		if !reflect.DeepEqual(expected, actual) {
			t.Fatalf("unexpected value, got: %v, want: %v", actual, expected)
		}
	})
	t.Run("response", func(t *testing.T) {
		input := xGetFeatureListResponse{
			XMLName:     xml.Name{Local: "u:X_GetFeatureListResponse"},
			Xmlns:       []xml.Attr{{Name: xml.Name{Local: "xmlns:u"}, Value: "urn:schemas-upnp-org:service:ContentDirectory:1"}},
			FeatureList: xmltypes.CommaSeparatedStrings([]string{"foo", "bar"}),
		}
		expected := `<u:X_GetFeatureListResponse xmlns:u="urn:schemas-upnp-org:service:ContentDirectory:1"><FeatureList>foo,bar</FeatureList></u:X_GetFeatureListResponse>`

		actual, err := xml.Marshal(input)
		if err != nil {
			t.Fatalf("could not marshal: %v", err)
		}

		if string(actual) != expected {
			t.Fatalf("unexpected value, got: %v, want: %v", string(actual), expected)
		}
	})
}
