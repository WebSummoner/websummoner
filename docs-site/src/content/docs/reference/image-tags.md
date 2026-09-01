---
title: Image tags and versioning
description: How browser image tags work — floating line tags, immutable full-version pins, and how each browser numbers its releases.
---

Every browser image is published under three tag levels, following the same
convention you may know from the official PostgreSQL or Redis images: a
short, human-friendly tag that moves with the release line, and a full
version tag that never changes.

| Tag level | Example | Meaning |
| --- | --- | --- |
| Line (floats) | `chrome:152`, `firefox:154`, `yandex:26.6`, `brave:1.94` | Always the latest patch of that release line. Rebuilt automatically whenever the vendor ships a patch. |
| Major.minor (alias) | `chrome:152.0`, `firefox:154.0` | Same image as the line tag; kept so existing scripts keep working. Skipped when the line already is major.minor (`yandex:26.6`, `brave:1.94`). |
| Full version (pinned) | `chrome:152.0.7977.64`, `firefox:154.0.1`, `brave:1.94.117` | The exact version the vendor published. Never rebuilt or repointed — pin this in CI for reproducible runs. |

```bash
docker pull websummoner/chrome:152            # floats to the latest 152.x
docker pull websummoner/chrome:152.0.7977.64  # this exact build, forever
```

## What each browser's "line" means

Browsers do not share one numbering scheme, so the line is the shortest
prefix that identifies the release users actually track:

| Browser | Line | Full version looks like | Notes |
| --- | --- | --- | --- |
| Chrome | `152` | `152.0.7977.64` | Chrome-for-Testing milestone; chromedriver matches the exact build |
| Firefox | `154` | `154.0`, then `154.0.1` | Mozilla never publishes `.0.0`; geckodriver is version-independent |
| Edge | `152` | `152.0.4191.53` | msedgedriver matches the exact build |
| Opera | `135` | `135.0.5973.66` | driver resolved from the Chromium line Opera ships (Opera N = Chromium N+16), falling back to the newest `operadriver` — see [Opera](/reference/browser-images/#opera) |
| Yandex Browser | `26.6` | `26.6.1.1083` | the line itself is major.minor |
| Brave | `1.94` | `1.94.117` | the line itself is major.minor; chromedriver matches the embedded Chromium |
| Safari | — | `2.52.6` | full WebKitGTK version only — each release is a substantially different engine, so nothing floats |

Unlike PostgreSQL, browsers do not publish a predictable version grid —
patches appear irregularly (`154.0`, `154.0.1`, sometimes `154.1.0`).
Asking for a version the vendor never published, such as
`firefox:154.0.0`, simply does not exist; the build fails instead of
silently falling back to something else.

## Coverage starts with the versions below

WebSummoner publishes browser images starting from the versions that were
current at its first release (September 2026): Chrome 152, Firefox 154, Edge
152, Opera 135, Yandex Browser 26.6, Brave 1.94 and WebKitGTK 2.52 — every
patch of those lines and of all future lines. **Legacy versions are not
provided**: the Selenoid-era images (Chrome 48–128, Firefox 3–128, …) are
not built, restored or planned. Old browsers stop receiving security
fixes, vendors delete the installation artifacts that image builds need,
and no modern test suite depends on them — so a rebuild would be insecure
and often impossible. If a test suite still pins such a version, keep the
old image it already has; those remain pullable from their original
registries.

There is also no `latest` tag by design: a tag that silently switches
browser *major* versions is rarely what a test suite wants. Pick a line
(`chrome:152`) or pin the full version.

## Checking what a floating tag resolves to

Every image carries labels recording the exact browser build and the
bundled driver — useful in CI logs to prove what actually ran:

```bash
docker image inspect websummoner/firefox:154 --format '{{json .Config.Labels}}'
# {"browser":"firefox:154.0.1~build1","driver":"geckodriver:0.37.1",...}
```

## Floating tags and drivers

Chromium-based browsers require the driver to match the browser build
(chromedriver and msedgedriver to the exact version, Brave's chromedriver
to the embedded Chromium line). Opera is the exception: its own driver
lags the Chromium it ships, so the image pins the newest `operadriver`
rather than an exactly-matching chromedriver — see
[Opera](/reference/browser-images/#opera). Floating tags are
therefore only safe because every rebuild re-pins the driver to the
matching version at the same time. A full-version tag always carries the
driver it was built and tested with.

If you need byte-for-byte reproducible test runs, pin the full version in
your CI. Once published, a full tag is never modified — Docker Hub itself
is the archive, even after the vendor stops hosting that patch.

## VNC is built into every image

There are no separate VNC images and no `vnc_*` tags. Every image ships
with Xvfb, fluxbox and x11vnc, and starts the VNC server at runtime when
asked to:

- Through the hub: add `enableVNC: true` to the session capabilities. The
  browser container then accepts VNC connections on port 5900 (password
  `websummoner`) and the screen is viewable in a browser at
  `http://<hub>:4444/vnc/<session-id>`.
- Standalone: run the image with `-e ENABLE_VNC=true -p 5900:5900` and
  connect any VNC client.

Users migrating from Selenoid should point `browsers.json` at the base
image name (`websummoner/chrome` instead of `selenoid/vnc_chrome`) and
keep the `enableVNC` capability — the behavior is identical.

## Requesting versions in sessions

The `version` capability matches browsers.json keys by prefix, so any of
`"152"`, `"152.0"` or the full `"152.0.7977.64"` selects the image when
the key covers it. An exact key always wins; otherwise the most specific
(longest) matching key is used — see
[Browsers configuration](/reference/browsers-config/#how-version-matching-works)
for the matching rules.
