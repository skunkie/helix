// SPDX-FileCopyrightText: 2020 Ethel Morgan
// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package contentdirectory

import (
	"context"
	"encoding/xml"
	"fmt"

	"github.com/ethulhu/helix/soap"
	"github.com/ethulhu/helix/upnpav"
	"github.com/ethulhu/helix/upnpav/contentdirectory/search"
	"github.com/ethulhu/helix/xmltypes"
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
		panic(fmt.Sprintf("could not marshal SOAP request: %v", err))
	}

	rsp, err := c.Call(ctx, string(Version1), method, req)
	if err != nil {
		return upnpav.MaybeError(err)
	}
	return xml.Unmarshal(rsp, output)
}

func (c *client) SearchCapabilities(ctx context.Context) ([]string, error) {
	req := getSearchCapabilitiesRequest{}
	rsp := getSearchCapabilitiesResponse{}
	if err := c.call(ctx, getSearchCapabilities, req, &rsp); err != nil {
		return nil, fmt.Errorf("could not get search capabilities: %w", err)
	}
	return rsp.Capabilities, nil
}
func (c *client) SortCapabilities(ctx context.Context) ([]string, error) {
	req := getSortCapabilitiesRequest{}
	rsp := getSortCapabilitiesResponse{}
	if err := c.call(ctx, getSortCapabilities, req, &rsp); err != nil {
		return nil, fmt.Errorf("could not get sort capabilities: %w", err)
	}
	return rsp.Capabilities, nil
}
func (c *client) SystemUpdateID(ctx context.Context) (uint, error) {
	req := getSystemUpdateIDRequest{}
	rsp := getSystemUpdateIDResponse{}
	if err := c.call(ctx, getSystemUpdateID, req, &rsp); err != nil {
		return 0, err
	}
	return rsp.SystemUpdateID, nil
}
func (c *client) XGetFeatureList(ctx context.Context) ([]string, error) {
	req := xGetFeatureListRequest{}
	rsp := xGetFeatureListResponse{}
	if err := c.call(ctx, xGetFeatureList, req, &rsp); err != nil {
		return nil, err
	}
	return rsp.FeatureList, nil
}

func (c *client) BrowseMetadata(ctx context.Context, object upnpav.ObjectID, sortCriteria xmltypes.CommaSeparatedStrings) (*upnpav.DIDLLite, error) {
	didl, _, err := c.browse(ctx, browseMetadata, object, 0, 0, sortCriteria)
	return didl, err
}
func (c *client) BrowseChildren(ctx context.Context, object upnpav.ObjectID, startingIndex, requestedCount uint, sortCriteria xmltypes.CommaSeparatedStrings) (*upnpav.DIDLLite, uint, error) {
	return c.browse(ctx, browseChildren, object, startingIndex, requestedCount, sortCriteria)
}
func (c *client) browse(ctx context.Context, bf browseFlag, object upnpav.ObjectID, startingIndex, requestedCount uint, sortCriteria xmltypes.CommaSeparatedStrings) (*upnpav.DIDLLite, uint, error) {
	req := browseRequest{
		Object:         object,
		BrowseFlag:     bf,
		Filter:         xmltypes.CommaSeparatedStrings{"*"},
		StartingIndex:  startingIndex,
		RequestedCount: requestedCount,
		SortCriteria:   sortCriteria,
	}

	rsp := browseResponse{}
	if err := c.call(ctx, browse, req, &rsp); err != nil {
		return nil, 0, fmt.Errorf("could not perform Browse request: %w", err)
	}
	return &rsp.Result.DIDLLite, rsp.TotalMatches, nil
}

func (c *client) Search(ctx context.Context, container upnpav.ObjectID, criteria search.Criteria, startingIndex, requestedCount uint, sortCriteria xmltypes.CommaSeparatedStrings) (*upnpav.DIDLLite, uint, error) {
	req := searchRequest{
		Container:      container,
		Filter:         xmltypes.CommaSeparatedStrings{"*"},
		SearchCriteria: criteria.String(),
		StartingIndex:  startingIndex,
		RequestedCount: requestedCount,
		SortCriteria:   sortCriteria,
	}

	rsp := searchResponse{}
	if err := c.call(ctx, searchA, req, &rsp); err != nil {
		return nil, 0, fmt.Errorf("could not perform Search request: %w", err)
	}
	return &rsp.Result.DIDLLite, rsp.TotalMatches, nil
}
