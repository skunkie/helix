// SPDX-FileCopyrightText: 2020 Ethel Morgan
// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package fileserver

import (
	"net/url"
	"path/filepath"
	"strings"

	"github.com/ethulhu/helix/upnpav"
	"github.com/ethulhu/helix/upnpav/contentdirectory"
)

func pathForObjectID(basePath string, id upnpav.ObjectID) (string, bool) {
	if id == contentdirectory.Root {
		return basePath, true
	}

	decoded, err := url.QueryUnescape(string(id))
	if err != nil {
		return "", false
	}

	maybePath := filepath.Clean(filepath.Join(basePath, filepath.FromSlash(decoded)))
	relPath, err := filepath.Rel(basePath, maybePath)
	if err != nil || relPath == ".." || filepath.IsAbs(relPath) ||
		strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", false
	}
	return maybePath, true
}

func objectIDForPath(basePath, p string) upnpav.ObjectID {
	if relPath, err := filepath.Rel(basePath, p); err == nil && relPath != "." {
		return upnpav.ObjectID(url.QueryEscape(relPath))
	}
	return contentdirectory.Root
}

func parentIDForPath(basePath, p string) upnpav.ObjectID {
	id := objectIDForPath(basePath, p)
	if id == contentdirectory.Root {
		return upnpav.ObjectID("-1")
	}
	return objectIDForPath(basePath, filepath.Dir(p))
}
