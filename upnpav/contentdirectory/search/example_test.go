// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package search_test

import (
	"fmt"
	"strings"

	"github.com/ethulhu/helix/upnpav"
	"github.com/ethulhu/helix/upnpav/contentdirectory/search"
)

// Release names separate words with dots, so a renderer searching for
// "sintel 2010" misses "Sintel.2010.1080p" under the default substring test.
// A ContainsFunc can treat those separators as spaces on both sides for titles
// only, which keeps every plain substring match.
func ExampleMatchesWith() {
	separators := strings.NewReplacer(".", " ", "_", " ", "-", " ")
	titleWords := func(property, value, operand string) bool {
		if property != "dc:title" {
			return search.ContainsFold(property, value, operand)
		}
		return search.ContainsFold(property, separators.Replace(value), separators.Replace(operand))
	}

	item := upnpav.Item{ID: "item.1", Class: upnpav.VideoItem, Title: "Sintel.2010.1080p"}
	criteria, err := search.Parse(`upnp:class derivedfrom "object.item.videoItem" and dc:title contains "sintel 2010"`)
	if err != nil {
		panic(err)
	}

	fmt.Println(search.Matches(item, criteria))
	fmt.Println(search.MatchesWith(item, criteria, titleWords))
	// Output:
	// false
	// true
}
