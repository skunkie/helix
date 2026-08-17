// SPDX-FileCopyrightText: 2020 Ethel Morgan
// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package media

import (
	"mime"
	"path"
	"strings"
)

func init() {
	types := map[string]string{
		".aac":  "audio/aac",
		".avi":  "video/x-msvideo",
		".flac": "audio/flac",
		".flv":  "video/x-flv",
		".m2ts": "video/mp2t",
		".m4a":  "audio/mp4",
		".m4v":  "video/mp4",
		".mkv":  "video/x-matroska",
		".mov":  "video/quicktime",
		".mp3":  "audio/mpeg",
		".mp4":  "video/mp4",
		".ogg":  "audio/ogg",
		".ts":   "video/mp2t",
		".wav":  "audio/wav",
		".webm": "video/webm",
		".wmv":  "video/x-ms-wmv",
	}
	for ext, mimeType := range types {
		_ = mime.AddExtensionType(ext, mimeType)
	}
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
