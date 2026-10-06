// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package fileserver

import (
	"strings"

	"github.com/ethulhu/helix/upnpav/contentdirectory/search"
)

// titleSeparators are the characters file names commonly put between words.
var titleSeparators = strings.NewReplacer(".", " ", "_", " ", "-", " ")

// titleContains decides `contains` for titles with word separators treated as
// spaces, because titles fall back to file names such as "Sintel.2010.1080p"
// that a renderer's search box would query as "sintel 2010". Other properties
// keep a plain case-insensitive substring test.
func titleContains(property, value, operand string) bool {
	if property != "dc:title" {
		return search.ContainsFold(property, value, operand)
	}
	return search.ContainsFold(property, titleSeparators.Replace(value), titleSeparators.Replace(operand))
}

func matches(obj interface{}, criteria search.Criteria) bool {
	return search.MatchesWith(obj, criteria, titleContains)
}
