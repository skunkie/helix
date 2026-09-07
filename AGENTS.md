<!--
SPDX-FileCopyrightText: 2026 TorrPlay

SPDX-License-Identifier: MIT
-->

# Repository Agent Instructions

## Commit Messages & History Hygiene

- **Subject Formatting**: Keep the subject line concise, in the imperative mood, starting with a lowercase verb (e.g., `add`, `fix`, `implement`, `extract`, `replace`, `update`), and without a trailing period. Do not use Conventional Commit prefixes (`feat:`, `fix:`) or scopes.
- **Body Formatting**: For non-trivial commits, add a body separated by a blank line. Write in plain prose paragraphs rather than bulleted lists (`-` or `*`). Wrap lines at standard widths (~72 characters).
- **Body Sentences**: Start each sentence in the body with a capital letter (often active verbs like `Add`, `Fix`, `Update`, `Ensure`, `Wire`) and end with a complete period.
- **Content Focus**: Describe observable behavior, design and architectural rationale, protocol/schema compatibility details (e.g., UPnP, DLNA, SOAP), and relevant test coverage (e.g., `Add unit tests for ...`). Do not narrate file-by-file edits.
- **Trivial Commits**: Minor, self-explanatory changes (e.g., simple log verbosity adjustments, single dependency replacements) may omit the body.
- **Atomic Commits**: Treat one cohesive change and its supporting refactors and tests as one commit, even when it touches multiple packages. Split changes only when they are independently meaningful and leave the repository correct at each boundary.
- **Timestamp Symmetry**: When rebasing, amending, or squashing commits, ensure `GIT_COMMITTER_DATE` matches `GIT_AUTHOR_DATE`.
- **Atomic Buildability**: Ensure every commit compiles and verifies cleanly.

Example:

```text
synchronize control loop state and propagate transport query errors

Protect playback loop fields with a mutex and pass point-in-time
snapshots of transport device, queue, and elapsed time to enactment
routines. Propagate transport query errors from PositionInfo and
MediaInfo instead of masking them as successful zero states. Validate
that seek positions are non-negative. Add tests for concurrent loop
access and transport error propagation.
```

## Build, Verification & Quality Standards

- **Go Toolchain**: Requires Go 1.24+.
- **Build Binaries**: Verify that all commands compile:
  ```bash
  go build ./cmd/...
  ```
- **Containerized Build**: Verify release build recipes when modifying `Makefile` or `Dockerfile.build`:
  ```bash
  make build
  ```
- **Test Suite**: Run all package tests:
  ```bash
  go test -count=1 ./...
  ```
- **Static Analysis & Linting**: Run `golangci-lint` to check code quality and formatting. Zero lint findings are permitted:
  ```bash
  golangci-lint run
  ```
- **Formatting**: Ensure all Go code is formatted with standard `gofmt -s`.
- **Pre-commit Checks**: Preserve trailing newline and clean whitespace according to `.pre-commit-config.yaml`.

## Licensing & Copyright Compliance

- **REUSE Specification**: All source, asset, test, and configuration files must comply with the REUSE specification.
- **SPDX Attribution for Fork Contributions**: New text files authored for TorrPlay and existing text files that receive substantial, copyrightable TorrPlay contributions must include a TorrPlay copyright notice using the actual contribution year or inclusive range of contribution years. For a modified upstream Go file with TorrPlay contributions spanning 2025 and 2026, retain the upstream notice alongside the TorrPlay notice:
  ```go
  // SPDX-FileCopyrightText: 2020 Ethel Morgan
  // SPDX-FileCopyrightText: 2025-2026 TorrPlay
  //
  // SPDX-License-Identifier: MIT
  ```
  Do not add a TorrPlay notice for solely mechanical or trivial edits such as formatting, typo corrections, or simple renames. Use comment syntax appropriate for the file format (`<!-- ... -->` for HTML/XML/Markdown, `# ...` for YAML/Makefiles/Shell).
- **Non-Commentable Files**: Record licensing and all applicable copyright notices for binary, generated, or otherwise non-commentable files with `REUSE.toml` annotations.
- **Upstream Copyrights**: Preserve all existing upstream copyright notices, including those for Ethel Morgan and Benedict Harcourt. Adding a TorrPlay notice must not replace an upstream notice.

## Core Architectural & Concurrency Invariants

- **Mutex Synchronization & Granularity**:
  - Protect shared mutable state (such as `controlpoint.Loop`, `controlpoint.Queue`, and device caches) using explicit `sync.Mutex` or `sync.RWMutex` locks.
  - Never hold locks across blocking network calls, HTTP requests, or channel operations (`select`, `done.Wait()`).
  - Take point-in-time snapshots under the lock and pass copies to long-running or enactment routines.
  - When resetting or clearing maps, reinitialize them before subsequent writes to prevent nil-map panics. Nil slices are acceptable when an empty slice has the intended semantics.
- **Context Propagation & Prompt Cancellation**:
  - Accept and propagate `context.Context` through all network queries, SSDP searches, HTTP requests, and SOAP calls.
  - Cancel timeout contexts promptly (`defer cancel()`) upon completion to prevent timer and goroutine leaks.
- **Resource Cleanup & Response Body Lifetime**:
  - Always close HTTP response bodies (`defer resp.Body.Close()`) and drain unread bytes when reusing connections.
  - Mark intentionally ignored cleanup errors explicitly (e.g. `_ = listener.Close()`, `_ = w.Flush()`).
- **Network Interface & IP Resolution**:
  - Support all RFC 1918 private IPv4 subnets (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`) via `netutil.SuitableIP`.
  - Do not hardcode loopback addresses (`127.0.0.1`) for network discovery, SSDP advertisements, or media server presentation URLs.
- **Filesystem Security**:
  - Prevent directory traversal in fileserver ContentDirectory implementations: clean paths using `filepath.Clean` and verify that resolved paths reside strictly within the root directory bounds.

## UPnP, DLNA & SOAP Protocol Standards

- **SSDP Discovery & Advertising**:
  - Perform multicast discovery on `239.255.255.250:1900` over HTTPU/HTTPMU.
  - Respect `MX` header response deadlines and handle `ssdp:alive` / `ssdp:byebye` announcement lifecycles.
- **SOAP & SCPD Envelope Handling**:
  - Support dynamic XML namespace prefixes in incoming SOAP requests and envelope wrappers.
  - Accept unqualified SOAP fault codes and escape XML fault descriptions properly.
  - **XML Zero Comparison Caveat**: Go's `xml.Unmarshal` always populates `XMLName`. Never compare response structs against zero values (`resp == Response{}`) to detect empty responses; verify payload fields directly.
- **GENA Eventing**:
  - Handle `SUBSCRIBE` and `UNSUBSCRIBE` requests with callback URLs parsed from `<...>` angle brackets.
  - Generate subscription identifiers (`SID`) using the `uuid:<uuid>` format.
  - Parse subscription timeouts safely against integer duration overflow (`Second-<integer>`).
  - Send event notifications using the `urn:schemas-upnp-org:event-1-0` namespace with `<e:propertyset>` and `<e:property>`, initiating subscriptions with sequence number `SEQ: 0`.
- **DIDL-Lite Metadata & ContentDirectory**:
  - Use standard DIDL-Lite namespaces (`urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/`, `dc`, `upnp`, `dlna`).
  - Serialize DIDL-Lite elements with correct attributes (`res` attributes: `protocolInfo`, `duration`, `resolution`, `size`, `bitrate`).
  - Support pagination parameters (`StartingIndex`, `RequestedCount`, returning `TotalMatches` and `NumberReturned`).
  - Support ascending (`+`) and descending (`-`) sort criteria on item and container properties.
  - Tokenize and evaluate search criteria using the AST evaluator in `upnpav/contentdirectory/search`.
- **AVTransport & ConnectionManager**:
  - Track playback states (`STOPPED`, `PLAYING`, `PAUSED_PLAYBACK`, `TRANSITIONING`).
  - Preserve subsecond precision in seek durations (`REL_TIME`).
  - Format ProtocolInfo as `http-get:*:<mime-type>:*` and support protocol negotiation.
  - Provide `X_MS_MediaReceiverRegistrar` support (`IsAuthorized`, `IsValid`) for Windows and DLNA renderer compatibility.

## Web Frontend Standards (Helix Player)

- **Static Asset Embedding**: Embed player web assets using `//go:embed static` in `cmd/helix-player/`.
- **Vanilla Modern JavaScript**: Use dependency-free ES modules and modern browser APIs without heavyweight build steps or bundlers.
- **Streaming Proxy Context**: Propagate incoming HTTP client request contexts through the media streaming reverse proxy so client disconnects terminate upstream streams immediately.
