// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package fileserver

import (
	"github.com/ethulhu/helix/upnpav/contentdirectory/search"
)

func matches(obj interface{}, criteria search.Criteria) bool {
	return search.Matches(obj, criteria)
}
