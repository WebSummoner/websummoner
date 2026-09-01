---
title: Recommended Docker settings
description: Storage drivers, limits and network tuning for a WebSummoner host.
---

These are the settings that matter most for a WebSummoner host, distilled
from running it in production clusters.

## Version and storage

- Use a recent Docker release on Linux kernel 4.x+ — older combinations have
  known bugs.
- Prefer the [OverlayFS](https://en.wikipedia.org/wiki/OverlayFS) storage
  driver. AUFS is fast but can orphan containers; never use Device Mapper —
  it is very slow. Check what you are on with:

  ```bash
  docker info | grep Storage
  ```

## Choosing `-limit`

The total number of simultaneous browser containers (`-limit`) is bound by
your hardware. A good starting point from experience:

```text
-limit ≈ 1.5–2.0 × number of CPU cores
```

## Bridged networking fix

On some Docker installations random Selenium commands fail with timeouts —
low `docker0` bridge performance is the usual culprit. Setting the bridge
MAC address to the same value as `eth0` fixes it:

```bash
ifconfig | grep eth0        # note HWaddr, e.g. 00:25:90:eb:fb:3e
ip link set docker0 address 00:25:90:eb:fb:3e
```

Make it permanent in `/etc/network/interfaces`:

```ini
iface docker0 inet static
    # ...
    post-up ip link set docker0 address 00:25:90:eb:fb:3e
```

## Per-container resources

Limit memory and CPU per browser container with the `-mem` and `-cpu` flags
(values in
[Docker format](https://docs.docker.com/engine/containers/resource_constraints/)):

```bash
./websummoner -mem 128m -cpu 1.5
```

## Docker client environment

WebSummoner uses the same client library as the `docker` CLI, so standard
variables like `DOCKER_API_VERSION` and `DOCKER_CERT_PATH` apply — full list
in the [Docker CLI environment
reference](https://docs.docker.com/reference/cli/docker/). If your daemon is
reachable over TCP instead of the local socket, follow the official
[remote access guide](https://docs.docker.com/engine/daemon/remote-access/)
to secure it first. If you see:

```text
[SERVICE_STARTUP_FAILED] ... client is newer than server
(client API version: 1.30, server API version: 1.24)
```

pin the API version to what your daemon supports:

```bash
docker run -e DOCKER_API_VERSION=1.24 -d --name websummoner \
  -p 4444:4444 \
  -v /etc/websummoner:/etc/websummoner:ro \
  -v /var/run/docker.sock:/var/run/docker.sock \
  websummoner/websummoner:latest-release
```
