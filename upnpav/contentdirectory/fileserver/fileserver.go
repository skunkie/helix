// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

// Package fileserver is a basic "serve a directory" ContentDirectory handler.
package fileserver

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ethulhu/helix/media"
	"github.com/ethulhu/helix/upnpav"
	"github.com/ethulhu/helix/upnpav/contentdirectory"
	"github.com/ethulhu/helix/upnpav/contentdirectory/search"
	"github.com/ethulhu/helix/xmltypes"

	log "github.com/sirupsen/logrus"
)

type (
	ContentDirectory struct {
		basePath string
		baseURL  *url.URL

		metadataCache media.MetadataCache

		mu             sync.RWMutex
		systemUpdateID uint
	}

	Features struct {
		XMLName        xml.Name `xml:"Features"`
		Xmlns          string   `xml:"xmlns,attr"`
		XmlnsXSI       string   `xml:"xmlns:xsi,attr"`
		SchemaLocation string   `xml:"xsi:schemaLocation,attr"`
		Feature        Feature  `xml:"Feature"`
	}

	Feature struct {
		Name       string      `xml:"name,attr"`
		Version    int         `xml:"version,attr"`
		Containers []Container `xml:"container"`
	}

	Container struct {
		ID   string `xml:"id,attr"`
		Type string `xml:"type,attr"`
	}
)

func NewContentDirectory(basePath, baseURL string, metadataCache media.MetadataCache) (*ContentDirectory, error) {
	maybeURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("could not parse base URL: %w", err)
	}

	absPath, err := filepath.Abs(basePath)
	if err != nil {
		return nil, fmt.Errorf("could not get absolute path: %w", err)
	}

	cd := &ContentDirectory{
		basePath: absPath,
		baseURL:  maybeURL,

		metadataCache:  metadataCache,
		systemUpdateID: uint(time.Now().Unix()),
	}

	go func() {
		fields := log.Fields{"path": absPath}
		log.WithFields(fields).Info("warming metadata cache")

		start := time.Now()
		metadataCache.Warm(absPath)
		fields["duration"] = time.Since(start)

		log.WithFields(fields).Info("finished warming metadata cache")
	}()

	return cd, nil
}

func (cd *ContentDirectory) XGetFeatureList(_ context.Context) ([]string, error) {
	features := Features{
		Xmlns:          "urn:schemas-upnp-org:av:avs",
		XmlnsXSI:       "http://www.w3.org/2001/XMLSchema-instance",
		SchemaLocation: "urn:schemas-upnp-org:av:avs http://www.upnp.org/schemas/av/avs.xsd",
		Feature: Feature{
			Name:    "samsung.com_BASICVIEW",
			Version: 1,
			Containers: []Container{
				{ID: "0", Type: "object.item.audioItem"},
				{ID: "0", Type: "object.item.videoItem"},
				{ID: "0", Type: "object.item.imageItem"},
			},
		},
	}

	bytes, err := xml.Marshal(features)
	if err != nil {
		return nil, err
	}
	return []string{string(bytes)}, nil
}

func (cd *ContentDirectory) BrowseMetadata(_ context.Context, id upnpav.ObjectID, _ xmltypes.CommaSeparatedStrings) (*upnpav.DIDLLite, error) {
	fields := log.Fields{
		"method": "BrowseMetadata",
		"object": id,
	}

	p, ok := pathForObjectID(cd.basePath, id)
	if !ok {
		log.WithFields(fields).Error("bad path")
		return nil, contentdirectory.ErrNoSuchObject
	}

	fi, err := os.Stat(p)
	if errors.Is(err, os.ErrNotExist) {
		log.WithFields(fields).Info("path does not exist")
		return nil, contentdirectory.ErrNoSuchObject
	}
	if err != nil {
		fields["error"] = err
		log.WithFields(fields).Warning("could not stat path")
		return nil, upnpav.ErrActionFailed
	}

	if fi.IsDir() {
		container, err := cd.containerFromPath(p)
		if err != nil {
			fields["error"] = err
			log.WithFields(fields).Warning("could not describe container from path")
			return nil, upnpav.ErrActionFailed
		}
		return &upnpav.DIDLLite{Containers: []upnpav.Container{container}}, nil
	}

	if !media.IsAudioOrVideo(p) {
		log.WithFields(fields).Warning("item exists but is not a media item")
		return nil, contentdirectory.ErrNoSuchObject
	}

	items, err := cd.itemsForPaths(p)
	if err != nil {
		fields["error"] = err
		log.WithFields(fields).Warning("could not describe item from path")
		return nil, upnpav.ErrActionFailed
	}
	return &upnpav.DIDLLite{Items: items}, nil
}

func (cd *ContentDirectory) BrowseChildren(_ context.Context, parent upnpav.ObjectID, sortCriteria xmltypes.CommaSeparatedStrings) (*upnpav.DIDLLite, error) {
	fields := log.Fields{
		"method": "BrowseChildren",
		"object": parent,
	}

	p, ok := pathForObjectID(cd.basePath, parent)
	if !ok {
		log.WithFields(fields).Error("bad path")
		return nil, contentdirectory.ErrNoSuchObject
	}

	fi, err := os.Stat(p)
	if errors.Is(err, os.ErrNotExist) {
		log.WithFields(fields).Info("path does not exist")
		return nil, contentdirectory.ErrNoSuchObject
	}
	if err != nil {
		fields["error"] = err
		log.WithFields(fields).Warning("could not stat path")
		return nil, upnpav.ErrActionFailed
	}

	if !fi.IsDir() {
		log.WithFields(fields).Info("not a directory")
		return nil, nil
	}

	didllite := &upnpav.DIDLLite{}

	fs, err := os.ReadDir(p)
	if err != nil {
		fields["error"] = err
		log.WithFields(fields).Error("could not list directory")
		return didllite, upnpav.ErrActionFailed
	}

	var itemPaths []string
	for _, fi := range fs {
		if strings.HasPrefix(fi.Name(), ".") {
			continue
		}

		if !fi.IsDir() {
			if media.IsAudioOrVideo(fi.Name()) {
				itemPaths = append(itemPaths, path.Join(p, fi.Name()))
			}
			continue
		}

		container, err := cd.containerFromPath(path.Join(p, fi.Name()))
		if err != nil {
			fields["error"] = err
			log.WithFields(fields).Warning("could not create container from path")
			continue
		}
		didllite.Containers = append(didllite.Containers, container)
	}

	items, err := cd.itemsForPaths(itemPaths...)
	if err != nil {
		fields["error"] = err
		log.WithFields(fields).Warning("could not create items from paths")
	}
	didllite.Items = items

	for _, criteria := range sortCriteria {
		if criteria == "dc:title" {
			sort.SliceStable(didllite.Items, func(i, j int) bool {
				return didllite.Items[i].Title < didllite.Items[j].Title
			})
		}
	}

	return didllite, nil
}

func (cd *ContentDirectory) SearchCapabilities(_ context.Context) ([]string, error) {
	return []string{"dc:title"}, nil
}
func (cd *ContentDirectory) SortCapabilities(_ context.Context) ([]string, error) {
	return []string{"dc:title"}, nil
}
func (cd *ContentDirectory) SystemUpdateID(_ context.Context) (uint, error) {
	cd.mu.RLock()
	defer cd.mu.RUnlock()
	return cd.systemUpdateID, nil
}

func (cd *ContentDirectory) IncrementSystemUpdateID() {
	cd.mu.Lock()
	defer cd.mu.Unlock()
	cd.systemUpdateID++
}

func (cd *ContentDirectory) Search(ctx context.Context, id upnpav.ObjectID, criteria search.Criteria) (*upnpav.DIDLLite, error) {
	results := &upnpav.DIDLLite{}
	walkFn := func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			return nil
		}
		if !media.IsAudioOrVideo(p) {
			return nil
		}

		items, err := cd.itemsForPaths(p)
		if err != nil {
			return nil
		}
		item := items[0]
		if matches(item, criteria) {
			results.Items = append(results.Items, item)
		}
		return nil
	}

	p, ok := pathForObjectID(cd.basePath, id)
	if !ok {
		return nil, contentdirectory.ErrNoSuchObject
	}

	if err := filepath.Walk(p, walkFn); err != nil {
		return nil, err
	}
	return results, nil
}

func (cd *ContentDirectory) containerFromPath(p string) (upnpav.Container, error) {
	container := upnpav.Container{
		ID:         objectIDForPath(cd.basePath, p),
		Parent:     parentIDForPath(cd.basePath, p),
		Class:      upnpav.StorageFolder,
		Restricted: true,
	}

	fi, err := os.Stat(p)
	if err != nil {
		return container, err
	}
	container.Title = fi.Name()
	container.Date = &upnpav.Date{Time: fi.ModTime()}

	fs, err := os.ReadDir(p)
	if err != nil {
		return container, err
	}
	container.ChildCount = len(fs)

	return container, nil
}

func (cd *ContentDirectory) itemsForPaths(paths ...string) ([]upnpav.Item, error) {
	coverArts := media.CoverArtForPaths(paths)
	metadatas := cd.metadataCache.MetadataForPaths(paths)

	titles := make([]string, len(metadatas))
	for i, md := range metadatas {
		titles[i] = md.Title
	}
	titles = trimCommonPrefix(titles)

	var items []upnpav.Item
	for i, p := range paths {
		md := metadatas[i]

		class, err := upnpav.ClassForMIMEType(md.MIMEType)
		if err != nil {
			panic(fmt.Sprintf("should only have audio or video MIME-Types, got %q for path %q", md.MIMEType, p))
		}

		var albumArtURIs []string
		for _, artPath := range coverArts[i] {
			albumArtURIs = append(albumArtURIs, cd.uri(artPath))
		}

		items = append(items, upnpav.Item{
			ID:           objectIDForPath(cd.basePath, p),
			Parent:       parentIDForPath(cd.basePath, p),
			Class:        class,
			Title:        titles[i],
			AlbumArtURIs: albumArtURIs,
			Resources: []upnpav.Resource{{
				URI:      cd.uri(p),
				Duration: &upnpav.Duration{Duration: md.Duration},
				ProtocolInfo: &upnpav.ProtocolInfo{
					Protocol:       upnpav.ProtocolHTTP,
					ContentFormat:  md.MIMEType,
					AdditionalInfo: upnpav.ContentFeatures,
				},
				SizeBytes: md.SizeBytes,
			}},
		})
	}

	return items, nil
}

func (cd *ContentDirectory) uri(p string) string {
	uri := *cd.baseURL
	relPath, _ := filepath.Rel(cd.basePath, p)
	uri.Path = path.Join(uri.Path, relPath)
	// TODO: figure out what's actually going wrong here.
	return strings.Replace((&uri).String(), "&", "%26", -1)
}

func trimCommonPrefix(ss []string) []string {
	if len(ss) == 0 {
		return ss
	}

	offset := strings.Index(ss[0], " - ")
	if offset == -1 {
		return ss
	}
	prefix := ss[0][0:offset] + " - "

	maybe := make([]string, len(ss))
	for i, s := range ss {
		if !strings.HasPrefix(s, prefix) {
			return ss
		}
		maybe[i] = strings.TrimPrefix(s, prefix)
	}
	return trimCommonPrefix(maybe)
}
