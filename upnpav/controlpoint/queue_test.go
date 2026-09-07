// SPDX-FileCopyrightText: 2020 Ethel Morgan
// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package controlpoint

import (
	"reflect"
	"sync"
	"testing"

	"github.com/ethulhu/helix/upnpav"
)

func TestTrackListAppendUpcoming(t *testing.T) {
	tracks := []upnpav.Item{
		{Title: "a"},
		{Title: "b"},
		{Title: "c"},
	}

	tl := NewTrackList()
	for _, track := range tracks {
		tl.Append(track)
	}

	for i, qi := range tl.Upcoming() {
		if !reflect.DeepEqual(qi.Item, tracks[i]) {
			t.Errorf("tl.Upcoming()[%d] == %+v, wanted %+v", i, qi.Item, tracks[i])
		}
	}
}

func TestTrackListAppendHistory(t *testing.T) {
	tracks := []upnpav.Item{
		{Title: "a"},
		{Title: "b"},
		{Title: "c"},
	}

	tl := NewTrackList()
	for _, track := range tracks {
		tl.Append(track)
	}

	for range tracks {
		tl.Skip()
	}

	for i, qi := range tl.History() {
		if !reflect.DeepEqual(qi.Item, tracks[i]) {
			t.Errorf("tl.Upcoming()[%d] == %+v, wanted %+v", i, qi.Item, tracks[i])
		}
	}
}

func TestTrackListRemoveNothing(t *testing.T) {
	tl := NewTrackList()
	tl.Remove(12)
}

func TestTrackListAddOneRemoveOne(t *testing.T) {
	tl := NewTrackList()
	id := tl.Append(upnpav.Item{Title: "a"})
	tl.Remove(id)

	if l := len(tl.Upcoming()); l != 0 {
		t.Errorf("len(tl.Upcoming()) == %d, expected 0", l)
	}
	if l := len(tl.History()); l != 0 {
		t.Errorf("len(tl.History()) == %d, expected 0", l)
	}
}
func TestTrackListAddOneRemoveNone(t *testing.T) {
	track := upnpav.Item{Title: "a"}

	tl := NewTrackList()
	id := tl.Append(track)
	tl.Remove(id + 1)

	if l := len(tl.Upcoming()); l != 1 {
		t.Errorf("len(tl.Upcoming()) == %d, expected 0", l)
	}
	if qi := tl.Upcoming()[0]; !reflect.DeepEqual(qi.Item, track) {
		t.Errorf("tl.Upcoming()[0] == %+v, expected %+v", qi.Item, track)
	}

	if l := len(tl.History()); l != 0 {
		t.Errorf("len(tl.History()) == %d, expected 0", l)
	}
}
func TestTrackListAddTwoRemoveFirst(t *testing.T) {
	track1 := upnpav.Item{Title: "a"}
	track2 := upnpav.Item{Title: "b"}

	tl := NewTrackList()
	id1 := tl.Append(track1)
	id2 := tl.Append(track2)

	tl.Remove(id1)

	want := []QueueItem{{ID: id2, Item: track2}}
	if !reflect.DeepEqual(tl.Upcoming(), want) {
		t.Errorf("tl.Upcoming() == %+v, expected %+v", tl.Upcoming(), want)
	}

	if l := len(tl.History()); l != 0 {
		t.Errorf("len(tl.History()) == %d, expected 0", l)
	}
}

func TestTrackListAppendAfterRemoveAll(t *testing.T) {
	tl := NewTrackList()
	tl.Append(upnpav.Item{Title: "first"})
	tl.RemoveAll()
	tl.Append(upnpav.Item{Title: "second"})

	if got := tl.Upcoming(); len(got) != 1 || got[0].Item.Title != "second" {
		t.Fatalf("Upcoming() = %+v, want one newly appended item", got)
	}
}

func TestTrackListConcurrentAccess(t *testing.T) {
	tl := NewTrackList()
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				id := tl.Append(upnpav.Item{Title: "track"})
				_, _ = tl.Current()
				_, _ = tl.Next()
				tl.Skip()
				_ = tl.SetCurrent(id)
				tl.Remove(id)
			}
		}()
	}
	wg.Wait()
}
