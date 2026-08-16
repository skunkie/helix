// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

package fileserver

import (
	"github.com/ethulhu/helix/upnpav/contentdirectory/search"
)

func matches(obj interface{}, criteria search.Criteria) bool {
	return search.Matches(obj, criteria)
}
