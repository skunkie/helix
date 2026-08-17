// SPDX-FileCopyrightText: 2020 Ethel Morgan
// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package upnpav

import (
	"errors"
	"fmt"
	"mime"
	"path/filepath"
	"strings"

	// Import media for side-effect initialization of audio/video MIME types in the global mime registry.
	_ "github.com/ethulhu/helix/media"
)

type (
	// ProtocolInfo is a UPnP AV ProtocolInfo string.
	ProtocolInfo struct {
		Protocol Protocol
		// Network should be "*" for http-get and rtsp-rtp-udp, but can have other values for other
		Network string
		// ContentFormat should be the MIME-type for http-get, or the RTP payload type for rtsp-rtp-udp.
		ContentFormat string
		// AdditionalInfo is frequently "*", but can be used by some formats, e.g. DLNA.ORG_PN extensions.
		AdditionalInfo string
	}

	Protocol string
)

const (
	ProtocolHTTP = Protocol("http-get")
	ProtocolRTSP = Protocol("rtsp-rtp-udp")
	// ContentFeatures is a DLNA extension to ProtocolInfo used by some renderers (e.g. Samsung TVs, WMP).
	ContentFeatures     = "DLNA.ORG_OP=01;DLNA.ORG_CI=0;DLNA.ORG_FLAGS=21700000000000000000000000000000"
	DefaultProtocolInfo = "http-get:*:image/jpeg:DLNA.ORG_PN=JPEG_TN,http-get:*:image/jpeg:DLNA.ORG_PN=JPEG_SM,http-get:*:image/jpeg:DLNA.ORG_PN=JPEG_MED,http-get:*:image/jpeg:DLNA.ORG_PN=JPEG_LRG,http-get:*:image/jpeg:DLNA.ORG_PN=JPEG_RES_H_V,http-get:*:image/png:DLNA.ORG_PN=PNG_TN,http-get:*:image/png:DLNA.ORG_PN=PNG_LRG,http-get:*:image/gif:DLNA.ORG_PN=GIF_LRG,http-get:*:audio/mpeg:DLNA.ORG_PN=MP3,http-get:*:audio/L16:DLNA.ORG_PN=LPCM,http-get:*:video/mpeg:DLNA.ORG_PN=AVC_TS_HD_24_AC3_ISO;SONY.COM_PN=AVC_TS_HD_24_AC3_ISO,http-get:*:video/vnd.dlna.mpeg-tts:DLNA.ORG_PN=AVC_TS_HD_24_AC3;SONY.COM_PN=AVC_TS_HD_24_AC3,http-get:*:video/vnd.dlna.mpeg-tts:DLNA.ORG_PN=AVC_TS_HD_24_AC3_T;SONY.COM_PN=AVC_TS_HD_24_AC3_T,http-get:*:video/vnd.dlna.mpeg-tts:DLNA.ORG_PN=MPEG_PS_PAL,http-get:*:video/vnd.dlna.mpeg-tts:DLNA.ORG_PN=MPEG_PS_NTSC,http-get:*:video/vnd.dlna.mpeg-tts:DLNA.ORG_PN=MPEG_TS_SD_50_L2_T,http-get:*:video/vnd.dlna.mpeg-tts:DLNA.ORG_PN=MPEG_TS_SD_60_L2_T,http-get:*:video/mpeg:DLNA.ORG_PN=MPEG_TS_SD_EU_ISO,http-get:*:video/vnd.dlna.mpeg-tts:DLNA.ORG_PN=MPEG_TS_SD_EU,http-get:*:video/vnd.dlna.mpeg-tts:DLNA.ORG_PN=MPEG_TS_SD_EU_T,http-get:*:video/vnd.dlna.mpeg-tts:DLNA.ORG_PN=MPEG_TS_SD_50_AC3_T,http-get:*:video/mpeg:DLNA.ORG_PN=MPEG_TS_HD_50_L2_ISO;SONY.COM_PN=HD2_50_ISO,http-get:*:video/vnd.dlna.mpeg-tts:DLNA.ORG_PN=MPEG_TS_SD_60_AC3_T,http-get:*:video/mpeg:DLNA.ORG_PN=MPEG_TS_HD_60_L2_ISO;SONY.COM_PN=HD2_60_ISO,http-get:*:video/vnd.dlna.mpeg-tts:DLNA.ORG_PN=MPEG_TS_HD_50_L2_T;SONY.COM_PN=HD2_50_T,http-get:*:video/vnd.dlna.mpeg-tts:DLNA.ORG_PN=MPEG_TS_HD_60_L2_T;SONY.COM_PN=HD2_60_T,http-get:*:video/mpeg:DLNA.ORG_PN=AVC_TS_HD_50_AC3_ISO;SONY.COM_PN=AVC_TS_HD_50_AC3_ISO,http-get:*:video/vnd.dlna.mpeg-tts:DLNA.ORG_PN=AVC_TS_HD_50_AC3;SONY.COM_PN=AVC_TS_HD_50_AC3,http-get:*:video/mpeg:DLNA.ORG_PN=AVC_TS_HD_60_AC3_ISO;SONY.COM_PN=AVC_TS_HD_60_AC3_ISO,http-get:*:video/vnd.dlna.mpeg-tts:DLNA.ORG_PN=AVC_TS_HD_60_AC3;SONY.COM_PN=AVC_TS_HD_60_AC3,http-get:*:video/vnd.dlna.mpeg-tts:DLNA.ORG_PN=AVC_TS_HD_50_AC3_T;SONY.COM_PN=AVC_TS_HD_50_AC3_T,http-get:*:video/vnd.dlna.mpeg-tts:DLNA.ORG_PN=AVC_TS_HD_60_AC3_T;SONY.COM_PN=AVC_TS_HD_60_AC3_T,http-get:*:video/x-mp2t-mphl-188:*,http-get:*:video/mp4:*,http-get:*:video/x-matroska:*,http-get:*:video/avi:*,http-get:*:video/x-msvideo:*,http-get:*:video/x-ms-wmv:*,http-get:*:video/quicktime:*,http-get:*:video/webm:*,http-get:*:video/*:*,http-get:*:audio/mp4:*,http-get:*:audio/flac:*,http-get:*:audio/wav:*,http-get:*:audio/aac:*,http-get:*:audio/*:*,http-get:*:image/*:*,http-get:*:text/srt:*,http-get:*:text/smi:*,http-get:*:text/ssa:*,http-get:*:*:*"
)

var (
	ErrUnknownMIMEType = errors.New("could not find valid MIME-type for URI")
)

func ParseProtocolInfo(raw string) (ProtocolInfo, error) {
	parts := strings.Split(raw, ":")
	if len(parts) != 4 {
		return ProtocolInfo{}, fmt.Errorf("ProtocolInfo must have 4 parts, found %v", len(parts))
	}
	return ProtocolInfo{
		Protocol:       Protocol(parts[0]),
		Network:        parts[1],
		ContentFormat:  parts[2],
		AdditionalInfo: parts[3],
	}, nil
}

func ProtocolInfoForURI(uri string) (*ProtocolInfo, error) {
	mimeType := mime.TypeByExtension(filepath.Ext(uri))
	if mimeType == "" {
		return nil, ErrUnknownMIMEType
	}
	return &ProtocolInfo{
		Protocol:       ProtocolHTTP,
		Network:        "*",
		ContentFormat:  mimeType,
		AdditionalInfo: "*",
	}, nil
}

func (p ProtocolInfo) String() string {
	network := "*"
	if p.Network != "" {
		network = p.Network
	}

	additionalInfo := "*"
	if p.AdditionalInfo != "" {
		additionalInfo = p.AdditionalInfo
	}

	return fmt.Sprintf("%s:%s:%s:%s", p.Protocol, network, p.ContentFormat, additionalInfo)
}

func (p ProtocolInfo) MarshalText() ([]byte, error) {
	return []byte(p.String()), nil
}
func (p *ProtocolInfo) UnmarshalText(raw []byte) error {
	pp, err := ParseProtocolInfo(string(raw))
	if err != nil {
		return err
	}
	*p = pp
	return nil
}
