---
title: Running manually
description: Start the WebSummoner binary or container yourself, with the flags and mounts you need.
---

This page is for when you want full control over how WebSummoner starts. If
you just want a working setup quickly, the [Quick start](/quick-start/) or
[Configuration Manager](https://github.com/WebSummoner/cm) is an easier path.
This guide assumes you are comfortable with the command line and basic Docker
commands.

## 1. Prepare the configuration

Create `config/browsers.json`:

```json
{
  "firefox": {
    "default": "154.0",
    "versions": {
      "154.0": {
        "image": "websummoner/firefox:155.0.0",
        "port": "4444",
        "path": "/"
      }
    }
  }
}
```

:::note
All maintained browser images serve the driver at `"/"`. Set `"path"` to
something else only for an image that runs a Selenium server instead of a
driver. See [Browsers configuration](/reference/browsers-config/) for the full
field reference.
:::

Then pull the browser image so the first session does not wait for a download:

```bash
docker pull websummoner/firefox:155.0.0
```

## 2. Start WebSummoner

### Option 1 — run the binary

Download the binary for your operating system from the
[releases page](https://github.com/WebSummoner/websummoner/releases/latest)
and save it as `websummoner` (or `websummoner.exe` on Windows). On Linux/macOS
add execution permissions with `chmod +x websummoner`, then run:

```bash
./websummoner
```

A successful start looks like:

```text
2026/08/30 21:23:43 Loading configuration files...
2026/08/30 21:23:43 Loaded configuration from [config/browsers.json]
...
2026/08/30 21:23:43 Listening on :4444
```

### Option 2 — run the container

With [Docker](https://docs.docker.com/engine/install/) installed you can
skip the binary and run WebSummoner itself as a container:

```bash
docker run -d                                \
  --name websummoner                         \
  -p 4444:4444                               \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $PWD/config/:/etc/websummoner/:ro       \
  websummoner/websummoner:latest-release
```

The Docker socket mount lets WebSummoner launch browser containers on the
same host.

Useful Docker references for the flags above:

- [Bind mounts](https://docs.docker.com/engine/storage/bind-mounts/) — how
  `-v /host/path:/container/path:ro` works, path formats per OS
- [`docker run` reference](https://docs.docker.com/reference/cli/docker/container/run/) —
  every flag used here
- [Docker networks](https://docs.docker.com/engine/network/) — bridge vs
  custom networks (`-container-network`)

## Ready-made browser images

Prebuilt images are maintained for several browsers:

- [Firefox](https://hub.docker.com/r/websummoner/firefox)
- [Google Chrome](https://hub.docker.com/r/websummoner/chrome)
- [Opera](https://hub.docker.com/r/websummoner/opera)

The complete list with versions and drivers is in
[Browser images](/reference/browser-images/). Build files live in the
[images repository](https://github.com/WebSummoner/images) — issues and
requests for new versions are welcome.

:::note[UTF-8 locales]
All images support UTF-8 locales, which matters if your tests save files with
non-ASCII names. Enable a specific locale through environment variables in
`browsers.json`:

```json
{
  "chrome": {
    "default": "152.0",
    "versions": {
      "152.0": {
        "image": "websummoner/chrome:152.0",
        "env": ["LANG=ru_RU.UTF-8", "LANGUAGE=ru:en", "LC_ALL=ru_RU.UTF-8"]
      }
    }
  }
}
```
:::

## Running on Windows

Everything works, but Docker volume mounts are the tricky part:

1. Replace backslashes (`\`) with forward slashes (`/`) and lowercase the
   drive letter — `C:\Users\admin` becomes `/c/Users/admin`.
2. Mount the Docker socket with **two** leading slashes:
   `-v //var/run/docker.sock:/var/run/docker.sock`

A typical startup (configuration at `C:\Users\admin\websummoner`):

```powershell
docker run -d --name websummoner `
  -p 4444:4444 `
  -v //var/run/docker.sock:/var/run/docker.sock `
  -v /c/Users/admin/websummoner:/etc/websummoner:ro `
  websummoner/websummoner:latest-release
```

:::tip
This PowerShell one-liner produces a mount-compatible `$PWD` (assumes you are
on the `C:` drive):

```powershell
$current = $PWD -replace "\\", "/" -replace "C", "c"
```
:::
