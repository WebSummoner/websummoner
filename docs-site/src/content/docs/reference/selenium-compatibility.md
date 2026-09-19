---
title: Selenium compatibility
description: Feature-by-feature parity between Selenium Grid and WebSummoner, and what is not implemented.
---

What works when you point a Selenium client at WebSummoner instead of Selenium
Grid, and what does not.

Checked against **Selenium 4.49.0** (released 9 September 2026). Selenium adds
capabilities faster than it changes the protocol, so this page is re-checked
each time we take a release; the date above is what it was last verified
against.

Legend: **Yes** works today · **Partial** works differently · **No** not
implemented. This page records what the current release does; it is not a
statement about what will or will not be built.

## WebDriver protocol

The hub proxies `/session/<id>/...` straight to the real driver, so the W3C
surface is whatever your driver implements — not something WebSummoner can lag
behind on.

| Area | WebSummoner |
| --- | --- |
| W3C WebDriver endpoints (navigation, elements, cookies, alerts, frames, actions) | Yes — proxied to the driver untouched |
| Print to PDF, screenshots, `executeScript` | Yes — driver-level |
| Timeouts, `unhandledPromptBehavior`, `strictFileInteractability` | Yes — driver-level |
| Legacy JSONWP clients (Selenium 2/3) | Yes — response is wrapped in the W3C envelope |

## Capabilities

| Capability | Purpose | WebSummoner |
| --- | --- | --- |
| `se:cdp`, `se:cdpVersion` | Chrome DevTools endpoint returned to the client | Yes — rewritten to the hub's own `/devtools/<id>/` address |
| `webSocketUrl` | Opt in to [WebDriver BiDi](https://www.selenium.dev/documentation/webdriver/bidi/) | **No** — see [BiDi](#webdriver-bidi) |
| `se:downloadsEnabled` | Enable Grid-managed downloads | **No** — WebSummoner has downloads under its own URL, see [Downloads](#downloads) |
| `se:vnc`, `se:noVncUrl` | Advertise a live-view URL in the returned capabilities | **No** — VNC works, it is simply not advertised |
| `selenoid:options` / `websummoner:options` | Video, VNC, logs, timeouts, container tuning | Yes — see [Capabilities](/reference/capabilities/) |

`se:cdp` is deliberately withheld for Firefox and Safari: neither has a CDP
endpoint, and advertising one makes clients fail their first WebSocket
handshake.

## Endpoints

| Endpoint | Grid | WebSummoner |
| --- | --- | --- |
| `/status` | Yes | Yes — plus queue, per-browser usage and live sessions |
| `/session` lifecycle | Yes | Yes |
| `GET /session/<id>/se/files` | Lists downloadable files | **No** — see below |
| `/graphql` | Grid's own query API | **No** — `/status` and `/metrics` cover similar ground |
| `/metrics` | Not built in | Yes — Prometheus, always on |
| `/video/`, `/logs/`, `/vnc/`, `/clipboard/`, `/download/` | Partly, and newer | Yes |

## Grid features

| Feature | WebSummoner |
| --- | --- |
| One container per session, discarded afterwards | Yes — the core model |
| Session queue with timeouts | Yes — and load-shedding, see [Session queue](/guides/session-queue/) |
| Video recording | Yes |
| Live view (VNC) | Yes |
| Session logs | Yes |
| Docker backend | Yes |
| Kubernetes backend | **No** |
| Node registration / multi-node routing | **No** in the hub — put [ggr](/reference/compare/) in front of several hubs |
| Relay to Appium or a cloud provider | **No** |

## Gaps in detail

### WebDriver BiDi

A client that sets `webSocketUrl: true` gets a socket address back from the
driver pointing at the browser container, which is unreachable from outside the
hub's network. WebSummoner does not rewrite it the way it already rewrites
`se:cdp`, so BiDi sessions do not connect.

This matters most for Firefox, where CDP is being retired upstream and BiDi is
the replacement, so Firefox currently has no low-level protocol through the
hub.

### Downloads

Selenium 4.8+ clients call `driver.get_downloadable_files()`, which issues
`GET /session/<id>/se/files` and expects the file names back under `names`.

WebSummoner has the same capability behind a different URL: a per-session
file server at `/download/<session-id>/<file>`, also reachable as
`/session/<id>/websummoner/download/<file>`. The feature is there, but stock
Selenium download code does not find it.

### Live-view capabilities

VNC works and is reachable at `/vnc/<session-id>`, but the URL is not returned
in the new-session capabilities, so tooling that discovers live view from
`se:vnc` or `se:noVncUrl` cannot find it automatically.

## Where WebSummoner goes further

| | |
| --- | --- |
| Per-session video, logs and VNC with no extra components | Built in rather than an add-on |
| Prometheus `/metrics` | Always available, no flag |
| Load-shedding admission control | [Session queue](/guides/session-queue/) |
| Browser discovery from image labels | [Discovery](/reference/browsers-config/#discovery-from-image-labels) |
| Memory footprint | A single Go binary rather than a JVM per node |

For a broader comparison including Moon, Playwright and the commercial clouds,
see [Comparing WebSummoner](/reference/compare/).
