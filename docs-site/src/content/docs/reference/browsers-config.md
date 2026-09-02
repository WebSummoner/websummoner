---
title: Browsers configuration
description: The browsers.json file — every field explained, with a realistic multi-version example.
---

WebSummoner reads a single JSON file that maps browser versions to Docker
images (or driver binaries). Pass its path with the `-conf` flag; the default
is `config/browsers.json` (inside the Docker image:
`/etc/websummoner/browsers.json`).

## A realistic example

Two browsers, several versions, sensible container tuning:

```json
{
  "chrome": {
    "default": "152.0",
    "versions": {
      "152.0": {
        "image": "websummoner/chrome:152.0",
        "port": "4444",
        "tmpfs": { "/tmp": "size=512m" }
      },
      "151.0": {
        "image": "websummoner/chrome:151.0",
        "port": "4444",
        "tmpfs": { "/tmp": "size=512m" }
      }
    }
  },
  "firefox": {
    "default": "154.0",
    "versions": {
      "154.0": {
        "image": "websummoner/firefox:155.0.0",
        "port": "4444",
        "path": "/",
        "tmpfs": { "/tmp": "size=512m" }
      }
    }
  }
}
```

## Top-level structure

| Field | Meaning |
| --- | --- |
| `<browser-name>` | Key matched against the Selenium `browserName` capability |
| `default` | Version used when the request carries no `version` capability |
| `versions` | Map of version name → version settings |

### How version matching works

Browser name and version are plain strings matched against the `browserName`
and `version` capabilities. An exact version key always wins; otherwise
WebSummoner falls back to **prefix matching**, picking the longest key that
*starts with* the requested version string. The keys usually mirror the
[image tag levels](/reference/image-tags/) — a line tag like `chrome:152`
works with the key `"152"`, a full pin with the full version string.

```text
browsers.json has  "152.0" and "152.0.7977.64"
request version =  "152"           → matches 152.0.7977.64 (longest prefix)
request version =  "152.0"         → matches 152.0           (exact key wins)
request version =  "152.1"         → no match  (no key starts with 152.1)
```

:::note[Browsers with special capability names]
Microsoft Edge must be listed under the browser name `MicrosoftEdge` in
`browsers.json` — not `edge` — because that is the `browserName` its driver
reports.
:::

## Version fields

| Field | Applies to | Description |
| --- | --- | --- |
| `image` | both | Docker image reference, **or** a command array for [standalone binaries](#standalone-driver-binaries) |
| `port` | containers | Real port the in-container process listens on |
| `path` | containers | Path where new sessions are created. All maintained images serve the driver at `/`; use `/wd/hub` only for an image that runs a Selenium server |
| `tmpfs` | containers | In-memory filesystems as `{ "mount": "size=512m" }` — browser caches on tmpfs are dramatically faster ([tmpfs docs](https://docs.docker.com/engine/storage/tmpfs/)) |
| `volumes` | containers | Host mounts as `["/host/dir:/container/dir:ro"]` ([bind mounts](https://docs.docker.com/engine/storage/bind-mounts/) · [volumes](https://docs.docker.com/engine/storage/volumes/)) |
| `env` | both | Environment variables as `["NAME=value"]` |
| `hosts` | containers | Extra `/etc/hosts` entries as `["hostname:ip"]` |
| `labels` | containers | Container labels as `{"key": "value"}` |
| `sysctl` | containers | Kernel parameters as `{"net.ipv4.tcp_timestamps": "2"}` |
| `shmSize` | containers | Shared memory size in bytes (default 256 MB — raise it if Chrome is unstable; background: [resource constraints](https://docs.docker.com/engine/containers/resource_constraints/)) |
| `cpu`, `mem` | containers | Per-container limits; can also be set globally with the `-cpu` / `-mem` flags |

:::tip
Images must already be pulled on the host — WebSummoner does not pull them
for you. [Configuration Manager](https://github.com/WebSummoner/cm) or the
`jq` one-liner below handle that.
:::

## Standalone driver binaries

In [drivers mode](/guides/without-docker/) (no Docker), the `image` field
holds a command in square brackets instead of a container reference:

```json
{
  "chrome": {
    "default": "152.0",
    "versions": {
      "152.0": {
        "image": ["/usr/bin/chromedriver", "--port=4444"],
        "port": "4444"
      }
    }
  }
}
```

## Syncing images from a file under version control

Keeping `browsers.json` in version control is a common pattern for
reproducible infrastructure. WebSummoner deliberately does not pull images
itself — under load, slow or failing pulls would make reload behavior
unpredictable — so use one of these instead.

**Option 1 — Configuration Manager** (best for fresh installs and CI jobs):

```bash
./cm websummoner start --browsers-json /path/to/your/browsers.json
```

**Option 2 — jq** (best for a running cluster, no downtime):

```bash
cat /path/to/browsers.json \
  | jq -r '..|.image?|strings' \
  | xargs -I{} docker pull {}
```

## Reloading and updating

- WebSummoner reloads `browsers.json` on `SIGHUP` — no restart needed.
  See [Reloading configuration](/reference/updating-configuration/).
- To move to new browser versions, edit the file and reload; images for the
  new versions must be pulled first.
