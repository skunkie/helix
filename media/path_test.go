// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

package media

import (
	"testing"
)

func TestIsAudioOrVideo(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "foo.mp3", want: true},
		{path: "foo.mp4", want: true},
		{path: "foo.mkv", want: true},
		{path: "foo.jpg", want: false},
		{path: "foo.txt", want: false},
		{path: "foo", want: false},
		{path: "foo.unknown", want: false},
		{path: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := IsAudioOrVideo(tt.path)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
