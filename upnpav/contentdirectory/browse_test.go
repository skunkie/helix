// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package contentdirectory

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
	"testing"

	"github.com/ethulhu/helix/upnpav"
	"github.com/ethulhu/helix/upnpav/contentdirectory/search"
	"github.com/ethulhu/helix/xmltypes"
)

func TestBrowsePaginationRequests(t *testing.T) {
	tests := []struct {
		name                 string
		startingIndex        uint
		requestedCount       uint
		expectedRspItems     int
		expectedFirstItemID  string
		expectedTotalMatches uint
	}{
		{
			name:                 "no pagination - return all",
			startingIndex:        0,
			requestedCount:       0,
			expectedRspItems:     5,
			expectedFirstItemID:  "item-0",
			expectedTotalMatches: 5,
		},
		{
			name:                 "first page",
			startingIndex:        0,
			requestedCount:       2,
			expectedRspItems:     2,
			expectedFirstItemID:  "item-0",
			expectedTotalMatches: 5,
		},
		{
			name:                 "second page",
			startingIndex:        2,
			requestedCount:       2,
			expectedRspItems:     2,
			expectedFirstItemID:  "item-2",
			expectedTotalMatches: 5,
		},
		{
			name:                 "last page partial",
			startingIndex:        4,
			requestedCount:       2,
			expectedRspItems:     1,
			expectedFirstItemID:  "item-4",
			expectedTotalMatches: 5,
		},
		{
			name:                 "large offset past end",
			startingIndex:        100,
			requestedCount:       10,
			expectedRspItems:     0,
			expectedFirstItemID:  "",
			expectedTotalMatches: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			items := make([]upnpav.Item, 5)
			for i := range items {
				items[i].ID = upnpav.ObjectID(fmt.Sprintf("item-%d", i))
			}
			fh := &countingFakeHandler{
				browseChildrenDIDLLite: &upnpav.DIDLLite{
					Items: items,
				},
			}

			client := NewClient(SOAPHandler{fh})

			didl, totalMatches, err := client.BrowseChildren(ctx, "1", tt.startingIndex, tt.requestedCount, nil)
			if err != nil {
				t.Fatalf("BrowseChildren(_, %d, %d) returned error: %v", tt.startingIndex, tt.requestedCount, err)
			}

			if fh.browseChildrenObject != "1" {
				t.Errorf("Interface.BrowseChildren called with object=%q, want %q", fh.browseChildrenObject, "1")
			}
			if fh.browseChildrenStartingIndex != tt.startingIndex {
				t.Errorf("Interface.BrowseChildren called with startingIndex=%d, want %d", fh.browseChildrenStartingIndex, tt.startingIndex)
			}
			if fh.browseChildrenRequestedCount != tt.requestedCount {
				t.Errorf("Interface.BrowseChildren called with requestedCount=%d, want %d", fh.browseChildrenRequestedCount, tt.requestedCount)
			}
			if len(didl.Items) != tt.expectedRspItems {
				t.Errorf("returned %d items, want %d", len(didl.Items), tt.expectedRspItems)
			}
			if tt.expectedRspItems > 0 && string(didl.Items[0].ID) != tt.expectedFirstItemID {
				t.Errorf("first item ID = %q, want %q", didl.Items[0].ID, tt.expectedFirstItemID)
			}
			if totalMatches != tt.expectedTotalMatches {
				t.Errorf("totalMatches=%d, want %d", totalMatches, tt.expectedTotalMatches)
			}
		})
	}
}

func TestBrowsePaginationXMLRoundTrip(t *testing.T) {
	fh := &countingFakeHandler{
		browseChildrenDIDLLite: &upnpav.DIDLLite{
			Items: []upnpav.Item{
				{ID: "item0", Title: "zero"},
				{ID: "item1", Title: "one"},
				{ID: "item2", Title: "two"},
				{ID: "item3", Title: "three"},
				{ID: "item4", Title: "four"},
			},
		},
	}
	h := SOAPHandler{Interface: fh}

	// Build a SOAP request with StartingIndex=2, RequestedCount=2
	browseXML := []byte(`
		<Browse xmlns="urn:schemas-upnp-org:service:ContentDirectory:1">
			<ObjectID>1</ObjectID>
			<BrowseFlag>BrowseDirectChildren</BrowseFlag>
			<Filter>*</Filter>
			<StartingIndex>2</StartingIndex>
			<RequestedCount>2</RequestedCount>
		</Browse>
	`)

	respBytes, err := h.Call(context.Background(), string(Version1), "Browse", browseXML)
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}

	if fh.browseChildrenStartingIndex != 2 {
		t.Errorf("StartingIndex=%d, want 2", fh.browseChildrenStartingIndex)
	}
	if fh.browseChildrenRequestedCount != 2 {
		t.Errorf("RequestedCount=%d, want 2", fh.browseChildrenRequestedCount)
	}

	var resp browseResponse
	if err := xml.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.NumberReturned != 2 {
		t.Errorf("NumberReturned=%d, want 2", resp.NumberReturned)
	}
	if resp.TotalMatches != 5 {
		t.Errorf("TotalMatches=%d, want 5", resp.TotalMatches)
	}
}

func TestBrowseMetadata(t *testing.T) {
	ctx := context.Background()
	fh := &countingFakeHandler{
		browseMetadataDIDLLite: &upnpav.DIDLLite{
			Containers: []upnpav.Container{{
				ID:         "1",
				Title:      "Root Container",
				ChildCount: 10,
			}},
		},
	}

	client := NewClient(SOAPHandler{fh})

	sortCrit := xmltypes.CommaSeparatedStrings{"+dc:title"}
	didl, err := client.BrowseMetadata(ctx, "1", sortCrit)
	if err != nil {
		t.Fatalf("BrowseMetadata returned error: %v", err)
	}

	if fh.browseMetadataObject != "1" {
		t.Errorf("Interface.BrowseMetadata called with object=%q, want %q", fh.browseMetadataObject, "1")
	}
	if len(fh.browseMetadataSortCriteria) != 1 || fh.browseMetadataSortCriteria[0] != "+dc:title" {
		t.Errorf("Interface.BrowseMetadata called with sortCriteria=%v, want %v", fh.browseMetadataSortCriteria, sortCrit)
	}
	if len(didl.Containers) != 1 || didl.Containers[0].ID != "1" {
		t.Errorf("expected 1 container with ID '1', got %v", didl.Containers)
	}
}

func TestBrowseChildrenSystemUpdateID(t *testing.T) {
	ctx := context.Background()
	fh := &countingFakeHandler{
		systemUpdateID: 42,
		browseChildrenDIDLLite: &upnpav.DIDLLite{
			Items: []upnpav.Item{{ID: "item"}},
		},
	}

	h := SOAPHandler{fh}

	browseXML := []byte(`
		<Browse xmlns="urn:schemas-upnp-org:service:ContentDirectory:1">
			<ObjectID>1</ObjectID>
			<BrowseFlag>BrowseDirectChildren</BrowseFlag>
			<Filter>*</Filter>
			<StartingIndex>0</StartingIndex>
			<RequestedCount>0</RequestedCount>
		</Browse>
	`)

	respBytes, err := h.Call(ctx, string(Version1), "Browse", browseXML)
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}

	var resp browseResponse
	if err := xml.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.UpdateID != 42 {
		t.Errorf("UpdateID=%d, want 42", resp.UpdateID)
	}
}

func TestBrowseRequestXMLParsing(t *testing.T) {
	tests := []struct {
		name            string
		xml             string
		wantStartingIdx uint
		wantReqCount    uint
		wantObject      string
		wantBrowseFlag  browseFlag
	}{
		{
			name:            "default values",
			xml:             `<u:Browse xmlns:u="urn:schemas-upnp-org:service:ContentDirectory:1"><ObjectID>1</ObjectID><BrowseFlag>BrowseDirectChildren</BrowseFlag><Filter>*</Filter></u:Browse>`,
			wantStartingIdx: 0,
			wantReqCount:    0,
			wantObject:      "1",
			wantBrowseFlag:  browseChildren,
		},
		{
			name:            "with pagination",
			xml:             `<u:Browse xmlns:u="urn:schemas-upnp-org:service:ContentDirectory:1"><ObjectID>2</ObjectID><BrowseFlag>BrowseDirectChildren</BrowseFlag><Filter>*</Filter><StartingIndex>10</StartingIndex><RequestedCount>5</RequestedCount></u:Browse>`,
			wantStartingIdx: 10,
			wantReqCount:    5,
			wantObject:      "2",
			wantBrowseFlag:  browseChildren,
		},
		{
			name:            "browse metadata",
			xml:             `<u:Browse xmlns:u="urn:schemas-upnp-org:service:ContentDirectory:1"><ObjectID>3</ObjectID><BrowseFlag>BrowseMetadata</BrowseFlag><Filter>*</Filter></u:Browse>`,
			wantStartingIdx: 0,
			wantReqCount:    0,
			wantObject:      "3",
			wantBrowseFlag:  browseMetadata,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req browseRequest
			if err := xml.Unmarshal([]byte(tt.xml), &req); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if req.Object != upnpav.ObjectID(tt.wantObject) {
				t.Errorf("Object=%q, want %q", req.Object, tt.wantObject)
			}
			if req.BrowseFlag != tt.wantBrowseFlag {
				t.Errorf("BrowseFlag=%q, want %q", req.BrowseFlag, tt.wantBrowseFlag)
			}
			if req.StartingIndex != tt.wantStartingIdx {
				t.Errorf("StartingIndex=%d, want %d", req.StartingIndex, tt.wantStartingIdx)
			}
			if req.RequestedCount != tt.wantReqCount {
				t.Errorf("RequestedCount=%d, want %d", req.RequestedCount, tt.wantReqCount)
			}
		})
	}
}

func TestBrowseResponseXMLMarshaling(t *testing.T) {
	resp := browseResponse{
		Result: upnpav.EncodedDIDLLite{DIDLLite: upnpav.DIDLLite{
			Items: []upnpav.Item{
				{ID: "item1", Title: "one"},
				{ID: "item2", Title: "two"},
			},
		}},
		NumberReturned: 2,
		TotalMatches:   10,
		UpdateID:       5,
	}

	respXML, err := xml.MarshalIndent(resp, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	xmlStr := string(respXML)
	if !strings.Contains(xmlStr, "NumberReturned>2</NumberReturned>") {
		t.Errorf("response missing NumberReturned=2:\n%s", xmlStr)
	}
	if !strings.Contains(xmlStr, "TotalMatches>10</TotalMatches>") {
		t.Errorf("response missing TotalMatches=10:\n%s", xmlStr)
	}
	if !strings.Contains(xmlStr, "UpdateID>5</UpdateID>") {
		t.Errorf("response missing UpdateID=5:\n%s", xmlStr)
	}
}

func TestBrowseChildrenWithSortCriteria(t *testing.T) {
	ctx := context.Background()
	fh := &countingFakeHandler{
		browseChildrenDIDLLite: &upnpav.DIDLLite{
			Items: []upnpav.Item{{ID: "item"}},
		},
	}

	client := NewClient(SOAPHandler{fh})

	sortCrit := xmltypes.CommaSeparatedStrings{"+dc:title", "-dc:date"}
	_, _, err := client.BrowseChildren(ctx, "1", 0, 0, sortCrit)
	if err != nil {
		t.Fatalf("BrowseChildren with sort criteria returned error: %v", err)
	}

	if fh.browseChildrenSortCriteria == nil {
		t.Error("sort criteria was not passed through")
	} else if len(fh.browseChildrenSortCriteria) != len(sortCrit) {
		t.Errorf("sort criteria length=%d, want %d", len(fh.browseChildrenSortCriteria), len(sortCrit))
	}
}

func TestBrowseTotalMatchesCalculation(t *testing.T) {
	ctx := context.Background()
	fh := &countingFakeHandler{
		browseChildrenTotalMatches: 50,
		browseChildrenDIDLLite: &upnpav.DIDLLite{
			Items: []upnpav.Item{
				{ID: "item1", Title: "one"},
				{ID: "item2", Title: "two"},
			},
		},
	}

	h := SOAPHandler{fh}

	browseXML := []byte(`
		<Browse xmlns="urn:schemas-upnp-org:service:ContentDirectory:1">
			<ObjectID>1</ObjectID>
			<BrowseFlag>BrowseDirectChildren</BrowseFlag>
			<Filter>*</Filter>
			<StartingIndex>0</StartingIndex>
			<RequestedCount>2</RequestedCount>
		</Browse>
	`)

	respBytes, err := h.Call(ctx, string(Version1), "Browse", browseXML)
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}

	var resp browseResponse
	if err := xml.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.NumberReturned != 2 {
		t.Errorf("NumberReturned=%d, want 2", resp.NumberReturned)
	}
	if resp.TotalMatches != 50 {
		t.Errorf("TotalMatches=%d, want 50", resp.TotalMatches)
	}
}

// countingFakeHandler captures the pagination params passed to BrowseChildren, BrowseMetadata, and Search.
type countingFakeHandler struct {
	browseChildrenObject         upnpav.ObjectID
	browseChildrenStartingIndex  uint
	browseChildrenRequestedCount uint
	browseChildrenSortCriteria   xmltypes.CommaSeparatedStrings
	browseChildrenDIDLLite       *upnpav.DIDLLite
	browseChildrenTotalMatches   uint

	browseMetadataObject       upnpav.ObjectID
	browseMetadataSortCriteria xmltypes.CommaSeparatedStrings
	browseMetadataDIDLLite     *upnpav.DIDLLite

	searchContainer      upnpav.ObjectID
	searchCriteria       search.Criteria
	searchStartingIndex  uint
	searchRequestedCount uint
	searchSortCriteria   xmltypes.CommaSeparatedStrings
	searchDIDLLite       *upnpav.DIDLLite
	searchTotalMatches   uint

	systemUpdateID     uint
	searchCapabilities []string
	sortCapabilities   []string
	xGetFeatureList    []string
}

func (f *countingFakeHandler) SearchCapabilities(_ context.Context) ([]string, error) {
	return f.searchCapabilities, nil
}
func (f *countingFakeHandler) SortCapabilities(_ context.Context) ([]string, error) {
	return f.sortCapabilities, nil
}
func (f *countingFakeHandler) SystemUpdateID(_ context.Context) (uint, error) {
	return f.systemUpdateID, nil
}
func (f *countingFakeHandler) XGetFeatureList(_ context.Context) ([]string, error) {
	return f.xGetFeatureList, nil
}
func (f *countingFakeHandler) BrowseMetadata(_ context.Context, id upnpav.ObjectID, sort xmltypes.CommaSeparatedStrings) (*upnpav.DIDLLite, error) {
	f.browseMetadataObject = id
	f.browseMetadataSortCriteria = sort
	return f.browseMetadataDIDLLite, nil
}
func (f *countingFakeHandler) BrowseChildren(_ context.Context, id upnpav.ObjectID, startingIndex, requestedCount uint, sort xmltypes.CommaSeparatedStrings) (*upnpav.DIDLLite, uint, error) {
	f.browseChildrenObject = id
	f.browseChildrenStartingIndex = startingIndex
	f.browseChildrenRequestedCount = requestedCount
	f.browseChildrenSortCriteria = sort
	total := f.browseChildrenTotalMatches
	if total == 0 && f.browseChildrenDIDLLite != nil {
		total = uint(len(f.browseChildrenDIDLLite.Containers) + len(f.browseChildrenDIDLLite.Items))
	}
	return f.browseChildrenDIDLLite.Paginate(startingIndex, requestedCount), total, nil
}
func (f *countingFakeHandler) Search(_ context.Context, id upnpav.ObjectID, criteria search.Criteria, startingIndex, requestedCount uint, sort xmltypes.CommaSeparatedStrings) (*upnpav.DIDLLite, uint, error) {
	f.searchContainer = id
	f.searchCriteria = criteria
	f.searchStartingIndex = startingIndex
	f.searchRequestedCount = requestedCount
	f.searchSortCriteria = sort
	total := f.searchTotalMatches
	if total == 0 && f.searchDIDLLite != nil {
		total = uint(len(f.searchDIDLLite.Containers) + len(f.searchDIDLLite.Items))
	}
	if f.searchDIDLLite != nil {
		return f.searchDIDLLite.Paginate(startingIndex, requestedCount), total, nil
	}
	return nil, total, nil
}

func TestSearchPagination(t *testing.T) {
	ctx := context.Background()
	items := make([]upnpav.Item, 10)
	for i := range items {
		items[i].ID = upnpav.ObjectID(fmt.Sprintf("item-%d", i))
		items[i].Title = fmt.Sprintf("test item %d", i)
	}

	fh := &countingFakeHandler{
		searchDIDLLite: &upnpav.DIDLLite{
			Items: items,
		},
		searchTotalMatches: 10,
	}

	h := SOAPHandler{fh}

	searchXML := []byte(`
		<Search xmlns="urn:schemas-upnp-org:service:ContentDirectory:1">
			<ContainerID>0</ContainerID>
			<SearchCriteria>(dc:title contains "test")</SearchCriteria>
			<Filter>*</Filter>
			<StartingIndex>5</StartingIndex>
			<RequestedCount>5</RequestedCount>
		</Search>
	`)

	respBytes, err := h.Call(ctx, string(Version1), "Search", searchXML)
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}

	if fh.searchStartingIndex != 5 {
		t.Errorf("searchStartingIndex=%d, want 5", fh.searchStartingIndex)
	}
	if fh.searchRequestedCount != 5 {
		t.Errorf("searchRequestedCount=%d, want 5", fh.searchRequestedCount)
	}

	var resp searchResponse
	if err := xml.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.NumberReturned != 5 {
		t.Errorf("NumberReturned=%d, want 5", resp.NumberReturned)
	}
	if resp.TotalMatches != 10 {
		t.Errorf("TotalMatches=%d, want 10", resp.TotalMatches)
	}
}
