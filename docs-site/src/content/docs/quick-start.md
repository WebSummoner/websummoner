---
title: Quick start
description: Run your first WebSummoner browser session in five minutes.
---

There are two ways to start WebSummoner: the plain **Docker path** (no extra
tools) and the **Configuration Manager path** (one binary that downloads and
configures everything for you). Both end at the same place: a Selenium endpoint
on port 4444.

## Option A — the Docker path

You need a recent [Docker Engine](https://docs.docker.com/engine/install/)
installation and nothing else.

### 1. Create a minimal `browsers.json`

This tells WebSummoner which browser images to launch. Save as `browsers.json`:

```json
{
  "chrome": {
    "default": "152",
    "versions": {
      "152": {
        "image": "websummoner/chrome:152",
        "port": "4444",
        "tmpfs": { "/tmp": "size=512m" }
      }
    }
  }
}
```

The `152` tag floats within the Chrome 152 release line — it always points
at the latest published patch. To pin an exact, never-changing build, use
the full version tag instead (see
[Image tags and versioning](/reference/image-tags/)).

### 2. Start the container

```bash
docker run -d --name websummoner -p 4444:4444 \
    -v $PWD/browsers.json:/etc/websummoner/browsers.json:ro \
    -v /var/run/docker.sock:/var/run/docker.sock \
    websummoner/websummoner:latest
```

The Docker socket mount is what allows WebSummoner to start browser
containers on the same host. New to bind mounts (`-v host:container`)? See
the official [bind mounts guide](https://docs.docker.com/engine/storage/bind-mounts/).

### 3. Check it is alive

Open <http://localhost:4444/status> — you should get a JSON document with
browser usage statistics (`"ready": true` means there is a free session slot).

## Option B — the Configuration Manager path

[Configuration Manager (**cm**)](https://github.com/WebSummoner/cm) is a small
helper binary that downloads the WebSummoner release, writes a `browsers.json`
with current browser versions and starts everything for you.

1. Download `cm` for your platform from the
   [releases page](https://github.com/WebSummoner/cm/releases/latest).
2. On Linux/macOS give it execution permissions:
   ```bash
   chmod +x cm
   ```
3. Start WebSummoner with VNC-capable browser images:
   ```bash
   ./cm websummoner start --vnc
   ```

:::warning[Running as root breaks things]
Do not run the command above with `sudo` — it leads to a broken installation.
Run as a regular user instead. On Linux you may need to add your user to the
`docker` group first:

```bash
sudo usermod -aG docker $USER
```
:::

The legacy `cm selenoid start` spelling still works — see
[Migrating from Selenoid](/migrating-from-selenoid/).

## Run your tests

Point your tests at WebSummoner exactly like you would at a regular Selenium
hub:

```text
http://localhost:4444/wd/hub
```

Any Selenium client and any framework (TestNG, JUnit, pytest, pytest-bot,
Nightwatch, WebdriverIO, Playwright's Selenium-compatible endpoint, …) works
unchanged.

## Optional: the WebSummoner UI

To watch live sessions, screens and logs in a browser, start
[WebSummoner UI](https://github.com/WebSummoner/websummoner-ui):

```bash
docker run -d --name websummoner-ui -p 8080:8080 \
    websummoner/websummoner-ui --websummoner-uri http://localhost:4444
```

Then open <http://localhost:8080/>.

## Where to go next

- [Running manually](/running-manually/) — flags, ports, and running the
  binary without Docker
- [Capabilities reference](/reference/capabilities/) — video, VNC, logs and
  everything else you can request per session
- [Browsers configuration](/reference/browsers-config/) — multi-version
  setups, memory limits, environment variables
