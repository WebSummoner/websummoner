---
title: Building browser images
description: Build your own browser images with the public images tool — exact versions, local packages, tests and push.
---

The [images repository](https://github.com/WebSummoner/images) ships a Go
tool that builds Docker images for every supported browser. Docker build
files are embedded in the binary, so one command produces a complete image.

## What is inside an image

Each image consists of 3 layers:

1. **Base layer** — everything every image needs: Xvfb, fonts for all major
   scripts (Latin, Cyrillic, CJK, Indic, Thai, Arabic, emoji), all UTF-8
   locales, timezone data, x11vnc and PulseAudio. Built once as
   `websummoner/browser-base`.
2. **Browser layer** — the browser binary, installed from the vendor's
   official package repository or from a `.deb` file.
3. **Driver layer** — the matching WebDriver binary (or the WebSummoner
   server for the Selenium-based Firefox variant).

## Building the tool

```bash
git clone https://github.com/WebSummoner/images.git
cd images
go build
./images --help
```

## Building an image

The `-b` flag always takes an **exact** package version — partial versions
are not resolved, and a version the vendor never published fails the build
instead of silently falling back to something else.

```bash
# Chrome (version is the google-chrome-stable package version)
./images chrome -b 152.0.7977.64-1 -t websummoner/chrome:152

# Firefox (version is the Mozilla apt package version)
./images firefox -b 154.0.1~build1 -t websummoner/firefox:154

# Edge (driver version must match the browser version)
./images edge -b 152.0.4191.53-1 -d 152.0.4191.53 -t websummoner/edge:152

# Opera (omit -d: the driver is resolved automatically, see the note below)
./images opera -b 135.0.5973.66-1 -t websummoner/opera:135

# Yandex (driver version is the YandexDriver Linux asset version)
./images yandex -b 26.6.1.1083-1 -d 26.6.1 -t websummoner/yandex:26.6
```

:::note[Opera drivers are resolved automatically]
Do not pass `-d` for Opera unless you have a specific reason. `operachromiumdriver`
release tags follow the **Chromium** version Opera is built on, not Opera's own
line — Opera N ships Chromium N+16 — so pinning by Opera's line fetches a driver
many majors too old, which refuses every session. The tool works out the Chromium
line, prefers Opera's own driver when one exists for it, and otherwise falls back
to the **newest published `operadriver`** — never a chromedriver, which starts
sessions but crashes the renderer on page-opened windows. The version check in
this driver family is only a warning, so a driver one line behind still works.
The image label records which one was used.
:::

Omitting `-d` (or passing `latest`) lets the tool pick the newest matching
driver. For Brave, the tool detects the embedded Chromium version from the
built image and downloads the matching chromedriver automatically.

To build from a local Debian package instead of a repository, pass the file
path — the file name must contain the full version, because the tool derives
the browser version from it:

```bash
./images firefox -b /path/to/firefox_154.0.1_amd64.deb -t websummoner/firefox:154
```

Add `--test` to run the container test suite after building (requires the
`websummoner-container-tests` checkout next to the images repository), and
`--push` to push the image to the registry.

## Tagging

Images published by RIADVICE follow the
[tag policy](/reference/image-tags/): a floating line tag, a major.minor
alias and an immutable full-version pin. When building for yourself, pass
several `-t` flags to apply the same convention.

## Browser channels

The `--channel` flag selects a non-stable package for browsers that publish
one in the same repository:

| Browser | Channel | Package |
| --- | --- | --- |
| Chrome | beta | `google-chrome-beta` |
| Chrome | dev | `google-chrome-unstable` |
| Edge | beta | `microsoft-edge-beta` |
| Edge | dev | `microsoft-edge-dev` |
| Opera | beta | `opera-beta` |
| Opera | dev | `opera-developer` |

Firefox channel variants previously relied on Ubuntu PPAs that no longer
exist; build from a `.deb` file of the desired channel instead.
