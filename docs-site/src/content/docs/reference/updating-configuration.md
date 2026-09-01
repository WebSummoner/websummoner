---
title: Updating configuration and browsers
description: Reload browsers.json without downtime, and roll browser versions forward.
---

Updating the browser list is the most frequent maintenance task. Two paths:
the fast one (restarts WebSummoner — fine for personal setups) and the
zero-downtime one (recommended for production clusters).

## The short way — `cm update`

```bash
./cm websummoner update
```

This downloads the latest WebSummoner release, pulls browser images,
regenerates `browsers.json` and restarts WebSummoner. Useful flags:

```bash
./cm websummoner update --vnc                  # VNC-capable images
./cm websummoner update --last-versions 5      # keep N last versions (default 2)
```

To pull images and regenerate the config **without restarting**:

```bash
./cm websummoner configure --vnc --last-versions 5
```

## The zero-downtime way — hot reload

1. Edit `browsers.json` (add/remove versions) and pull any new images, e.g.:

   ```bash
   docker pull websummoner/chrome:152.0
   ```

   The `jq` one-liner in
   [Browsers configuration](/reference/browsers-config/#syncing-images-from-a-file-under-version-control)
   pulls everything the file mentions.

2. Reload without restarting the process:

   ```bash
   kill -HUP <pid>                  # when running as a binary
   docker kill -s HUP websummoner   # when running as a container
   ```

   (Use only one of the two commands.)

3. Verify the reload took effect:

   ```bash
   curl -s http://example.com:4444/ping
   # {"uptime":"2m46s","lastReloadTime":"...","numRequests":42}
   ```

`/ping` returns `200 OK` when healthy, along with uptime, the last reload
time and the total number of processed session requests.

:::note
Removing a version from `browsers.json` does not kill already-running
sessions of that version — they run until deleted.
:::
