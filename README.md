<!--
SPDX-FileCopyrightText: 2026 TorrPlay

SPDX-License-Identifier: MIT
-->

# Helix

Helix is a collection of UPnP AV / DLNA media servers, web-based control points, and command-line diagnostics written in Go.

---

## Overview

Helix provides implementations of the UPnP AV 1.0 specifications, including:
- **SSDP Device Discovery & Advertising:** Multicast discovery (HTTPU/HTTPMU) and background advertisement lifecycle.
- **ContentDirectory Service:** Browse containers/items, search with query criteria, pagination (`StartingIndex`, `RequestedCount`), sorting, and DIDL-Lite XML serialization.
- **ConnectionManager Service:** ProtocolInfo negotiation, active connection tracking, and MIME-type classification.
- **Microsoft DLNA Extensions:** Built-in `X_MS_MediaReceiverRegistrar` service for authorization and discovery in Windows Explorer, Windows Media Player, and Xbox.
- **Web-based Control Point:** Queue and stream media from any UPnP ContentDirectory to any UPnP AVTransport renderer on the local network.

---

## Applications

### 1. `helix-directory`
A lightweight, standalone UPnP-AV ContentDirectory media server that serves audio and video files from a local directory tree.

* **Automatic SSDP discovery:** Broadcasts device availability to local UPnP renderers and control points.
* **In-memory metadata cache:** Fast responses with asynchronous cache warming.
* **Live filesystem monitoring:** Uses `fsnotify` to track file changes and increments the UPnP `SystemUpdateID`.

```bash
helix-directory -path /path/to/media -port 8080 -friendly-name "My Media Server"
```

### 2. `helix-player`
A web-based UPnP AV control point and media player. It discovers ContentDirectory servers and AVTransport renderers on your local network, allowing you to browse/search media libraries, manage a playback queue, and stream tracks to your devices.

```bash
helix-player -port 8081
```

Open `http://localhost:8081` in your browser to access the web interface.

### 3. `helix-directory-jackalope`
A UPnP-AV ContentDirectory server backed by a [Jackalope](https://go.eth.moe/jackalope) tag-based media database.

---

## CLI Utilities

Helix includes command-line tools for discovering, querying, and testing UPnP AV devices on your network:

| Utility | Description |
| :--- | :--- |
| **`list-devices`** | Discover and list all UPnP devices active on the local network via SSDP. |
| **`list-device-urls`** | Print presentation and control URLs for discovered UPnP devices. |
| **`list-content-directory`** | Browse containers and items from a ContentDirectory server (supports `-starting-index` and `-requested-count` pagination). |
| **`search-content-directory`** | Search a ContentDirectory server with UPnP search criteria queries (e.g. `(dc:title contains "song")`). |
| **`get-media-info`** | Fetch current media URI and metadata from an AVTransport service. |
| **`get-position-info`** | Query playback position, duration, and track info from an AVTransport device. |
| **`get-transport-info`** | Query playback state (PLAYING, PAUSED_PLAYBACK, STOPPED) from an AVTransport device. |
| **`get-search-capabilities`** | Print search capabilities supported by a ContentDirectory server. |
| **`get-protocol-info`** | Fetch protocol information and supported MIME types from a ConnectionManager service. |
| **`get-file-metadata`** | Extract audio/video metadata and MIME types from local media files. |

---

## Library Packages

The codebase is organized into modular Go packages:

* **[`upnp`](upnp/):** SSDP client/server, HTTPU multicast transport, device caching, and SCPD (Service Control Protocol Description) parsing.
* **[`upnpav`](upnpav/):** DIDL-Lite XML document building, parsing, and pagination (`DIDLLite.Paginate`), along with UPnP-AV types ([`Date`](upnpav/date.go), [`Duration`](upnpav/duration.go), [`URL`](upnpav/url.go), [`ProtocolInfo`](upnpav/protocolinfo.go), [`Resolution`](upnpav/resolution.go)).
* **[`upnpav/contentdirectory`](upnpav/contentdirectory/):** ContentDirectory service client and SOAP handler.
* **[`upnpav/contentdirectory/search`](upnpav/contentdirectory/search/):** Query tokenizer, parser, and [`search.Matches`](upnpav/contentdirectory/search/matches.go) evaluator for UPnP search expressions.
* **[`upnpav/contentdirectory/fileserver`](upnpav/contentdirectory/fileserver/):** Filesystem-backed ContentDirectory implementation.
* **[`upnpav/connectionmanager`](upnpav/connectionmanager/):** ConnectionManager service client, server, and SOAP handler.
* **[`upnpav/mediareceiverregistrar`](upnpav/mediareceiverregistrar/):** Microsoft DLNA authorization service (`X_MS_MediaReceiverRegistrar`).
* **[`upnpav/avtransport`](upnpav/avtransport/):** AVTransport client and SOAP handler.
* **[`upnpav/controlpoint`](upnpav/controlpoint/):** Continuous playback control loop and playlist track queue.
* **[`media`](media/):** Tag parsing (MP3, MP4, FLAC, Vorbis) and MIME type mapping.
* **[`xmltypes`](xmltypes/):** Specialized XML marshaling types (`IntBool`, `YesNoBool`, `CommaSeparatedStrings`).

---

## Installation & Build

### Prerequisites
* Go 1.24+

### Building All Binaries
```bash
# Build all command-line tools and servers
go build ./cmd/...

# Or install them to $GOPATH/bin
go install ./cmd/...
```

### Running Tests
```bash
go test -count=1 ./...
```

---

## License

MIT License. See [LICENSE](LICENSE) for details.
