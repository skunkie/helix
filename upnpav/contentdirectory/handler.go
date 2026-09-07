// SPDX-FileCopyrightText: 2020 Ethel Morgan
// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package contentdirectory

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/ethulhu/helix/logger"
	"github.com/ethulhu/helix/soap"
	"github.com/ethulhu/helix/upnpav"
	"github.com/ethulhu/helix/upnpav/contentdirectory/search"
	"github.com/ethulhu/helix/xmltypes"
)

type (
	SOAPHandler struct {
		Interface
	}
)

func (h SOAPHandler) Call(ctx context.Context, namespace, action string, in []byte) ([]byte, error) {
	if !strings.EqualFold(namespace, string(Version1)) {
		return nil, fmt.Errorf("invalid namespace: %q", namespace)
	}

	switch action {
	case getSearchCapabilities:
		return h.getSearchCapabilities(ctx, in)
	case getSortCapabilities:
		return h.getSortCapabilities(ctx, in)
	case getSystemUpdateID:
		return h.getSystemUpdateID(ctx, in)
	case browse:
		return h.browse(ctx, in)
	case searchA:
		return h.search(ctx, in)
	case xGetFeatureList:
		return h.xGetFeatureList(ctx, in)
	default:
		return nil, upnpav.ErrInvalidAction
	}
}

func (h SOAPHandler) getSearchCapabilities(ctx context.Context, in []byte) ([]byte, error) {
	req := getSearchCapabilitiesRequest{}
	if err := xml.Unmarshal(in, &req); err != nil {
		return nil, upnpav.ErrInvalidArgs
	}

	caps, err := h.SearchCapabilities(ctx)
	if err != nil {
		return nil, err
	}

	rsp := getSearchCapabilitiesResponse{
		Capabilities: caps,
	}

	prefix := soap.DetectPrefix(in, getSearchCapabilities)
	if prefix != "" {
		rsp.XMLName = xml.Name{Local: prefix + ":GetSearchCapabilitiesResponse"}
		rsp.Xmlns = []xml.Attr{{Name: xml.Name{Local: "xmlns:" + prefix}, Value: string(Version1)}}
	} else {
		rsp.XMLName = xml.Name{Space: string(Version1), Local: "GetSearchCapabilitiesResponse"}
	}

	return xml.Marshal(rsp)
}
func (h SOAPHandler) getSortCapabilities(ctx context.Context, in []byte) ([]byte, error) {
	req := getSortCapabilitiesRequest{}
	if err := xml.Unmarshal(in, &req); err != nil {
		return nil, upnpav.ErrInvalidArgs
	}

	caps, err := h.SortCapabilities(ctx)
	if err != nil {
		return nil, err
	}

	rsp := getSortCapabilitiesResponse{
		Capabilities: caps,
	}

	prefix := soap.DetectPrefix(in, getSortCapabilities)
	if prefix != "" {
		rsp.XMLName = xml.Name{Local: prefix + ":GetSortCapabilitiesResponse"}
		rsp.Xmlns = []xml.Attr{{Name: xml.Name{Local: "xmlns:" + prefix}, Value: string(Version1)}}
	} else {
		rsp.XMLName = xml.Name{Space: string(Version1), Local: "GetSortCapabilitiesResponse"}
	}

	return xml.Marshal(rsp)
}
func (h SOAPHandler) getSystemUpdateID(ctx context.Context, in []byte) ([]byte, error) {
	req := getSystemUpdateIDRequest{}
	if err := xml.Unmarshal(in, &req); err != nil {
		return nil, upnpav.ErrInvalidArgs
	}

	id, err := h.SystemUpdateID(ctx)
	if err != nil {
		return nil, err
	}

	rsp := getSystemUpdateIDResponse{
		SystemUpdateID: id,
	}

	prefix := soap.DetectPrefix(in, getSystemUpdateID)
	if prefix != "" {
		rsp.XMLName = xml.Name{Local: prefix + ":GetSystemUpdateIDResponse"}
		rsp.Xmlns = []xml.Attr{{Name: xml.Name{Local: "xmlns:" + prefix}, Value: string(Version1)}}
	} else {
		rsp.XMLName = xml.Name{Space: string(Version1), Local: "GetSystemUpdateIDResponse"}
	}

	return xml.Marshal(rsp)
}

func (h SOAPHandler) xGetFeatureList(ctx context.Context, in []byte) ([]byte, error) {
	req := xGetFeatureListRequest{}
	if err := xml.Unmarshal(in, &req); err != nil {
		return nil, upnpav.ErrInvalidArgs
	}

	featureList, err := h.XGetFeatureList(ctx)
	if err != nil {
		return nil, err
	}

	rsp := xGetFeatureListResponse{
		FeatureList: xmltypes.CommaSeparatedStrings(featureList),
	}

	prefix := soap.DetectPrefix(in, xGetFeatureList)
	if prefix != "" {
		rsp.XMLName = xml.Name{Local: prefix + ":X_GetFeatureListResponse"}
		rsp.Xmlns = []xml.Attr{{Name: xml.Name{Local: "xmlns:" + prefix}, Value: string(Version1)}}
	} else {
		rsp.XMLName = xml.Name{Space: string(Version1), Local: "X_GetFeatureListResponse"}
	}

	return xml.Marshal(rsp)
}

func (h SOAPHandler) browse(ctx context.Context, in []byte) ([]byte, error) {
	req := browseRequest{}
	log, _ := logger.FromContext(ctx)
	if err := xml.Unmarshal(in, &req); err != nil {
		log.WithError(err).Warning("could not unmarshal request")
		return nil, upnpav.ErrInvalidArgs
	}

	var err error
	var didllite *upnpav.DIDLLite
	var totalMatches uint
	switch req.BrowseFlag {
	case browseMetadata:
		didllite, err = h.BrowseMetadata(ctx, req.Object, req.SortCriteria)
		if didllite != nil {
			totalMatches = uint(len(didllite.Containers) + len(didllite.Items))
		}
	case browseChildren:
		didllite, totalMatches, err = h.BrowseChildren(ctx, req.Object, req.StartingIndex, req.RequestedCount, req.SortCriteria)
	default:
		return nil, upnpav.ErrInvalidArgs
	}
	if err != nil {
		return nil, err
	}

	rsp := browseResponse{}
	if didllite != nil {
		rsp.Result = upnpav.EncodedDIDLLite{DIDLLite: *didllite}
		rsp.NumberReturned = uint(len(didllite.Containers) + len(didllite.Items))
		rsp.TotalMatches = totalMatches
		updateID, err := h.SystemUpdateID(ctx)
		if err != nil {
			log.WithError(err).Warning("could not get system update ID")
		} else {
			rsp.UpdateID = updateID
		}
	}

	prefix := soap.DetectPrefix(in, browse)
	if prefix != "" {
		rsp.XMLName = xml.Name{Local: prefix + ":BrowseResponse"}
		rsp.Xmlns = []xml.Attr{{Name: xml.Name{Local: "xmlns:" + prefix}, Value: string(Version1)}}
	} else {
		rsp.XMLName = xml.Name{Space: string(Version1), Local: "BrowseResponse"}
	}

	return xml.Marshal(rsp)
}

func (h SOAPHandler) search(ctx context.Context, in []byte) ([]byte, error) {
	req := searchRequest{}
	log, _ := logger.FromContext(ctx)
	if err := xml.Unmarshal(in, &req); err != nil {
		log.WithError(err).Warning("could not unmarshal request")
		return nil, upnpav.ErrInvalidArgs
	}

	criteria, err := search.Parse(req.SearchCriteria)
	if err != nil {
		return nil, fmt.Errorf("could not parse search query: %w", err)
	}

	didllite, totalMatches, err := h.Search(ctx, req.Container, criteria, req.StartingIndex, req.RequestedCount, req.SortCriteria)
	if err != nil {
		return nil, err
	}

	rsp := searchResponse{}
	if didllite != nil {
		rsp.Result = upnpav.EncodedDIDLLite{DIDLLite: *didllite}
		rsp.NumberReturned = uint(len(didllite.Containers) + len(didllite.Items))
		rsp.TotalMatches = totalMatches
		updateID, err := h.SystemUpdateID(ctx)
		if err != nil {
			log.WithError(err).Warning("could not get system update ID")
		} else {
			rsp.UpdateID = updateID
		}
	}

	prefix := soap.DetectPrefix(in, searchA)
	if prefix != "" {
		rsp.XMLName = xml.Name{Local: prefix + ":SearchResponse"}
		rsp.Xmlns = []xml.Attr{{Name: xml.Name{Local: "xmlns:" + prefix}, Value: string(Version1)}}
	} else {
		rsp.XMLName = xml.Name{Space: string(Version1), Local: "SearchResponse"}
	}

	return xml.Marshal(rsp)
}
