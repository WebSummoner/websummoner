---
title: Browser images
description: Ready-made Docker images with browsers — where to find them and what is inside.
---

WebSummoner runs any image that speaks WebDriver, but you usually want the
maintained ones:

| Browser | Image | Notes |
| --- | --- | --- |
| Chrome | `websummoner/chrome` | Driver-based, `path: /` |
| Firefox | `websummoner/firefox` | Driver-based, `path: /` |
| Opera | `websummoner/opera` | Driver-based, `path: /` |
| Brave | `websummoner/brave` | Chromium-based, driver-based, `path: /` |
| Edge | `websummoner/edge` | Driver-based, `path: /` |
| Yandex Browser | `websummoner/yandex` | Driver-based, `path: /` |
| Safari (WebKit engine) | `websummoner/safari` | WebKitGTK build, `browserName: safari`, `path: /` |
| Video recorder | `websummoner/video-recorder` | Used automatically for `enableVideo` |

All images are free to use. Each browser is published under three tag
levels — a floating line tag, a major.minor alias and an immutable
full-version pin; see
[Image tags and versioning](/reference/image-tags/). Version coverage
starts from the releases that were current at WebSummoner's first
release (September 2026); legacy browser versions are not provided. For
example:

```bash
docker pull websummoner/chrome:152
docker pull websummoner/firefox:155
docker pull websummoner/edge:152
docker pull websummoner/opera:135
docker pull websummoner/yandex:26.6
docker pull websummoner/brave:1.94
```

## Browser-specific behaviour

Differences between browsers, and the configuration each image needs, are on
[Browser-specific behaviour](/reference/browser-notes/).

## Where things live

- Build files and per-browser documentation (exact versions, bundled
  drivers): the [images repository](https://github.com/WebSummoner/images).
- New browser versions are provisioned and published to Docker Hub by
  RIADVICE as they stabilize. Custom builds are possible with the public
  [`images` tool](https://github.com/WebSummoner/images#building-images).

## VNC support

VNC is built into every image: an `x11vnc` server streams the browser
screen whenever the session is started with the `enableVNC` capability
(or `ENABLE_VNC=true` when running the image standalone). It is required
for the live view in
[WebSummoner UI](https://github.com/WebSummoner/websummoner-ui). There are
no separate VNC images.

## Safari note

Real Safari only runs on macOS and iOS. `websummoner/safari` is built from
the official WebKitGTK releases — the same engine Safari uses — compiled
with its matching WebKitWebDriver, so behavior is functionally equivalent
to Safari, though fonts and pixel-perfect rendering can differ from the
macOS browser. The `browserName` capability is `safari`; versions follow
the WebKitGTK release numbering rather than Safari marketing numbers.
Safari is tagged with the full WebKitGTK version only (`safari:2.52.6`) —
there is no floating line tag, because each WebKitGTK release is a
substantially different engine:

```json
{
  "safari": {
    "default": "2.52.6",
    "versions": {
      "2.52.6": {
        "image": "websummoner/safari:2.52.6",
        "port": "4444",
        "path": "/"
      }
    }
  }
}
```

Testing the actual macOS Safari requires Apple hardware — WebSummoner can
drive it through the [standalone driver
binaries](/reference/browsers-config/#standalone-driver-binaries) form of
`browsers.json` on a Mac running `safaridriver`.

## Custom root certificates

On corporate networks the tested environment often uses TLS certificates
from a private root CA. The standard `acceptInsecureCerts` capability
ignores certificate errors, but does not help with HSTS. Instead, add your
root certificate to an image at container start with environment variables —
one variable per certificate, holding the Base64-encoded `cert.pem`:

```bash
CERT_CONTENTS=$(cat cert.pem | base64 -w0)   # macOS: base64
docker run -e ROOT_CA_MY_CERT="$CERT_CONTENTS" ... websummoner/chrome:152
```

The variable suffix (`MY_CERT` above) becomes the certificate name in the
browser certificate storage.

## Custom browser profile in Chrome

When launching Chrome with a custom profile directory, DevTools do not work
unless `BROWSER_PROFILE_DIR` is also set to the same directory:

```json
{
  "capabilities": {
    "alwaysMatch": {
      "browserName": "chrome",
      "goog:chromeOptions": {
        "args": ["user-data-dir=/profiles/custom.XYZ"]
      },
      "websummoner:options": {
        "env": ["BROWSER_PROFILE_DIR=/profiles/custom.XYZ"]
      }
    }
  }
}
```

## Audio

Every standard image runs a PulseAudio server in the container. Video
recordings therefore include an audio track automatically — see
[Recording audio](/guides/video-recording/#recording-audio).

## Custom images

Any Docker image works as long as it starts a WebDriver-compatible service on
a known port — see the `image`, `port` and `path` fields in
[Browsers configuration](/reference/browsers-config/). To build your own
browser images with the public `images` tool, see
[Building browser images](/reference/building-browser-images/).
