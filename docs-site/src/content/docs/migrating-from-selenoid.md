---
title: Migrating from Selenoid
description: What changes when you move from Aerokube Selenoid to WebSummoner — and what deliberately does not.
---

WebSummoner is a fork of [Aerokube Selenoid](https://github.com/aerokube/selenoid)
that keeps the protocol, configuration format and day-to-day operation intact.
Most migrations are a rename plus one path change. This page lists every
difference that can affect you.

## What stays the same

- **Selenium wire protocol** — same endpoint (`http://host:4444/wd/hub`), same
  session lifecycle, same API for video, logs, VNC, clipboard, downloads and
  DevTools proxying.
- **`browsers.json` format** — identical schema (image, port, path, tmpfs,
  volumes, env, hosts, shmSize, cpu, mem).
- **CLI flags** — same names and semantics (`-limit`, `-conf`, `-timeout`,
  `-video-output-dir`, …).
- **Status endpoint** — `/status`, `/ping`, `/wd/hub` and friends are
  unchanged.
- **GGR compatibility** — WebSummoner works as a GGR backend exactly like
  Selenoid did.

## What is renamed

| Selenoid | WebSummoner | Notes |
| --- | --- | --- |
| `selenoid` binary | `websummoner` | Release assets are named `websummoner_<os>_<arch>` |
| `aerokube/selenoid` image | `websummoner/websummoner` | On Docker Hub |
| `selenoid/chrome:128.0` etc. | `websummoner/chrome:152.0` etc. | New versions are published under the `websummoner` org |
| `selenoid/video-recorder` | `websummoner/video-recorder` | Default of `-video-recorder-image` |
| `cm selenoid start` | `cm websummoner start` | `cm selenoid` still works as an alias |
| `/etc/selenoid/browsers.json` | `/etc/websummoner/browsers.json` | **Inside the Docker image only** — see below |
| `/opt/selenoid/video/` | `/opt/websummoner/video/` | Default video dir inside the image |

## The one breaking change: in-image paths

The Docker image's entrypoint now reads its configuration from
`/etc/websummoner/browsers.json` and writes video to `/opt/websummoner/video`.
If you mounted volumes at the old paths, update the mount targets:

```bash
# before
-v $PWD/browsers.json:/etc/selenoid/browsers.json:ro
# after
-v $PWD/browsers.json:/etc/websummoner/browsers.json:ro
```

Everything you run outside the container (your own `-conf` flag, your own
`-video-output-dir`) is unaffected.

## Backwards-compatible wire extensions

Old clients keep working — both spellings are accepted, new ones win when both
are present:

| Thing | Preferred | Still accepted |
| --- | --- | --- |
| Capability namespace | `websummoner:options` | `selenoid:options` |
| No-wait header | `X-WebSummoner-No-Wait` | `X-Selenoid-No-Wait` |
| File-upload header | `X-WebSummoner-File` | `X-Selenoid-File` |
| Vendor proxy path | `/wd/hub/session/<id>/websummoner/...` | `/wd/hub/session/<id>/aerokube/...` |

Example — both of these create a session with VNC enabled:

```json
{ "browserName": "chrome", "websummoner:options": { "enableVNC": true } }
{ "browserName": "chrome", "selenoid:options": { "enableVNC": true } }
```

## Tooling compatibility

- **Selenoid UI** — use [WebSummoner UI](https://github.com/WebSummoner/websummoner-ui);
  it consumes the same status API.
- **GGR** — the maintained fork lives at
  [WebSummoner/ggr](https://github.com/WebSummoner/ggr).
- **Selenium clients** — no changes needed at all.

## Frequently asked migration questions

**Can I keep using my existing `selenoid/*` browser images?**
Yes. `browsers.json` accepts any image reference, including the historical
`selenoid/chrome:*` images that still exist on Docker Hub. New versions are
only published under `websummoner/*`.

**Does `-video-recorder-image` still default to the Selenoid recorder?**
No — it now defaults to `websummoner/video-recorder:latest-release`. Pass the
flag explicitly if you need the old image.

**Is the configuration file format versioned?**
No, it is the same JSON schema; your existing files work as-is.
