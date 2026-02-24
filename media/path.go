// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

package media

import (
	"mime"
	"path"
	"strings"
)

func init() {
	_ = mime.AddExtensionType(".mp3", "audio/mpeg")
	_ = mime.AddExtensionType(".mp4", "video/mp4")
	_ = mime.AddExtensionType(".mkv", "video/x-matroska")
}

func IsAudioOrVideo(p string) bool {
	ext := path.Ext(p)
	mimeType := mime.TypeByExtension(ext)
	return strings.HasPrefix(mimeType, "audio/") || strings.HasPrefix(mimeType, "video/")
}

func IsImage(p string) bool {
	ext := path.Ext(p)
	mimeType := mime.TypeByExtension(ext)
	return strings.HasPrefix(mimeType, "image/")
}
