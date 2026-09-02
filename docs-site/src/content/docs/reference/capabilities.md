---
title: Capabilities
description: Every WebSummoner capability in one table — video, VNC, logs, limits, networking and more.
---

Capabilities are per-session settings passed with your Selenium new-session
request. Any client that can send custom capabilities works; for clients that
only allow W3C-standard capabilities, use the `websummoner:options` protocol
extension (see [below](#passing-capabilities-as-protocol-extensions)).

## Quick reference

| Capability | Type | Default | What it does |
| --- | --- | --- | --- |
| `enableVNC` | boolean | `false` | Expose the live browser screen over WebSocket |
| `screenResolution` | string | `1920x1080x24` | Screen resolution, `<width>x<height>x<depth>` |
| `enableVideo` | boolean | `false` | Record the session to an H.264 video file |
| `videoName` | string | `<session-id>.mp4` | Output video file name (keep the extension; plain file name only, no folders) |
| `videoScreenSize` | string | screen size | Video resolution, e.g. `1024x768` (trims from top-left) |
| `videoFrameRate` | int | `12` | Video frames per second |
| `videoCodec` | string | `libx264` | ffmpeg codec name (lower CPU alternatives exist) |
| `enableAudio` | boolean | `true` | Include the audio track in recordings (set `false` for video-only) |

:::note[Protocol compatibility]
The legacy `version` capability and vendor-prefixed blocks (`websummoner:options`,
`selenoid:options`) are accepted from clients but automatically stripped before
the new-session request is forwarded to the browser container — modern
chromedriver rejects unknown top-level capabilities.
:::
| `enableLog` | boolean | `false` | Save session log to a file |
| `logName` | string | `<session-id>.log` | Log file name (keep the extension; plain file name only, no folders) |
| `name` | string | — | Human-readable test name, shown in the UI |
| `sessionTimeout` | duration | `-timeout` flag | Idle timeout, e.g. `30m` or `10s`; capped by `-max-timeout` |
| `timeZone` | string | host time zone | IANA zone, e.g. `Europe/Moscow` |
| `env` | array | — | Extra env vars for the browser container, `["K=V"]` |
| `applicationContainers` | array | — | Containers to link, `["name:alias"]` |
| `hostsEntries` | array | — | Extra `/etc/hosts` entries, `["host:ip"]` |
| `dnsServers` | array | Docker defaults | Custom DNS servers, `["192.168.0.1"]` |
| `additionalNetworks` | array | `-container-network` | Extra Docker networks to attach |
| `labels` | map | — | Container labels, `{"env": "testing"}` |
| `containerHostname` | string | container ID | Custom hostname inside the container |
| `s3KeyPattern` | string | `-s3-key-pattern` | Override the S3 upload key pattern |

## Session features

### Live browser screen — `enableVNC`

```json
{ "enableVNC": true }
```

Works with images that ship a VNC server (the *VNC* column of
[Browser images](/reference/browser-images/)). The screen is proxied to
`http://<host>:4444/vnc/<session-id>` as a WebSocket — open it via
[WebSummoner UI](https://github.com/WebSummoner/websummoner-ui) for a
point-and-click view.

### Custom screen resolution — `screenResolution`

```json
{ "screenResolution": "1280x1024x24" }
```

:::warning
This sets the **screen** resolution, not the browser window size. Browsers
have a default window size, so screenshots can be smaller than the screen.
Resize the window explicitly in your test. Because containers run headless
browsers in Xvfb without a window manager, `maximize` does not work — use
`setSize` instead.
:::

### Video recording — `enableVideo` and friends

```json
{
  "enableVideo": true,
  "videoName": "my-cool-video.mp4",
  "videoScreenSize": "1024x768",
  "videoFrameRate": 24,
  "videoCodec": "mpeg4",
  "enableAudio": false
}
```

`enableAudio` is `true` by default — recordings capture the browser's sound
(see [Recording audio](/guides/video-recording/#recording-audio)). Set it to
`false` explicitly when you want video-only recordings (privacy, CPU or file
size reasons).

Requires a video recorder image — WebSummoner defaults to
`websummoner/video-recorder:latest-release`, so recording works out of the
box with the standard images. See the
[Video recording guide](/guides/video-recording/) for retrieving and deleting
files, codecs and disk planning.

:::note
Always keep the `mp4` extension in `videoName`. Use a plain file name —
values containing `/`, `\` or `..` are rejected.
:::

### Session logs — `enableLog`, `logName`

```json
{ "enableLog": true, "logName": "my-cool-log.log" }
```

To save logs for **all** sessions without asking per test, start WebSummoner
with `-save-all-logs`. See [Session logs](/guides/session-logs/) for the
retrieval API — and keep the `log` extension in `logName`.

### Test name — `name`

```json
{ "name": "myCoolTestName" }
```

Shown per session in the UI; makes debugging parallel runs much easier. Also
added as a container label automatically.

## Session environment

### Idle timeout — `sessionTimeout`

```json
{ "sessionTimeout": "30m" }
```

Go duration format (`30m`, `10s`, `1h5m`); values above the `-max-timeout`
flag are clamped to it.

### Time zone — `timeZone`

```json
{ "timeZone": "Europe/Moscow" }
```

Any [IANA zone](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones).
Without it, containers inherit the WebSummoner host time zone.

### Environment variables — `env`

```json
{ "env": ["LANG=ru_RU.UTF-8", "LANGUAGE=ru:en", "LC_ALL=ru_RU.UTF-8"] }
```

Appended to the variables set in `browsers.json` — useful for locale tests.

## Networking

### Linked application containers — `applicationContainers`

```json
{ "applicationContainers": ["spring-application-main:my-cool-app", "spring-application-gateway"] }
```

Lets tests use URLs like `http://my-cool-app/` against app containers on the
same host.

### Hosts entries — `hostsEntries`

```json
{ "hostsEntries": ["example.com:192.168.0.1", "test.com:192.168.0.2"] }
```

Inserted **before** the entries from `browsers.json`, so capability entries
win on conflicts.

### DNS servers — `dnsServers`

```json
{ "dnsServers": ["192.168.0.1", "192.168.0.2"] }
```

Overrides the Docker daemon DNS defaults for this session's container.

### Additional networks — `additionalNetworks`

```json
{ "additionalNetworks": ["my-custom-net-1", "my-custom-net-2"] }
```

Containers always join the `-container-network` network; add more when the
tested application lives elsewhere.

## Metadata

### Container labels — `labels`

```json
{ "labels": { "environment": "testing", "build-number": "14353" } }
```

Useful in clusters to enrich centralized logs (environment, VCS revision,
build number). Overrides same-named labels from `browsers.json`.

### S3 key pattern — `s3KeyPattern`

```json
{ "s3KeyPattern": "$quota/$fileType$fileExtension" }
```

Overrides the `-s3-key-pattern` flag for this session. Supported placeholders
are listed in [Uploading files to S3](/guides/s3-upload/).

## Passing capabilities as protocol extensions

Some Selenium clients only accept W3C-standard capabilities. For those,
WebSummoner reads the WebDriver protocol-extensions block under the
`websummoner:options` key. These two requests are equivalent:

```json
{ "browserName": "firefox", "version": "155.0.0", "screenResolution": "1280x1024x24" }
```

```json
{
  "browserName": "firefox",
  "version": "154.0",
  "websummoner:options": { "screenResolution": "1280x1024x24" }
}
```

:::note[Backwards compatibility]
The legacy Selenoid `selenoid:options` key is still accepted. When both keys
are present, values from `websummoner:options` take precedence.
:::
