---
title: CLI flags
description: Every websummoner command-line flag, grouped by what it controls.
---

All flags go to the `websummoner` binary (or after the image name when using
`docker run`). Durations use Go format: `30s`, `5m`, `1h5m`.

## Core

| Flag | Default | Description |
| --- | --- | --- |
| `-listen` | `:4444` | Network address to accept connections on |
| `-conf` | `config/browsers.json` | Path to the [browsers configuration file](/reference/browsers-config/) |
| `-log-conf` | — | Path to the [container logging configuration file](/reference/logging-config/) |
| `-limit` | `5` | Maximum simultaneous browser containers |
| `-retry-count` | `1` | New-session attempts before giving up |
| `-disable-queue` | `false` | Disable the wait queue (fail instead of waiting when `-limit` is reached) |
| `-version` | — | Print version and exit |

The `/metrics` endpoint is always available (no flag needed) — see
[Usage statistics](/guides/usage-statistics/#prometheus-metrics).

## Timeouts

| Flag | Default | Description |
| --- | --- | --- |
| `-timeout` | `1m0s` | Session idle timeout |
| `-max-timeout` | `1h0m0s` | Upper bound for per-session `sessionTimeout` capability |
| `-session-attempt-timeout` | `30s` | Timeout for a single new-session attempt |
| `-session-delete-timeout` | `30s` | Timeout for deleting a session |
| `-service-startup-timeout` | `30s` | Timeout for a browser container/driver to start |
| `-graceful-period` | `5m0s` | Graceful shutdown period (finish live sessions) |

## Video and logs

| Flag | Default | Description |
| --- | --- | --- |
| `-video-output-dir` | `video` | Directory for recorded video files |
| `-video-recorder-image` | `websummoner/video-recorder:latest-release` | Image used to record video |
| `-log-output-dir` | — | Directory for saved session logs |
| `-save-all-logs` | `false` | Save logs for every session, ignoring capabilities |
| `-capture-driver-logs` | `false` | Add driver process logs to WebSummoner output (drivers mode only) |

## Docker

| Flag | Default | Description |
| --- | --- | --- |
| `-disable-docker` | `false` | Drivers mode — run browsers as local processes, [no Docker needed](/guides/without-docker/) |
| `-disable-privileged` | `false` | Start browser containers without `--privileged` |
| `-container-network` | `default` | Docker network for browser containers |
| `-cpu` | — | CPU limit per container as float, e.g. `0.2` or `1.0` |
| `-mem` | — | Memory limit per container, e.g. `128m` or `1g` |
| `-enable-file-upload` | `false` | Enable the [file upload](/guides/file-upload/) API |

## S3 upload

Only available in binaries built with S3 support (the release binaries and
the standard Docker image include it). See
[Uploading files to S3](/guides/s3-upload/) for a full walkthrough.

| Flag | Default | Description |
| --- | --- | --- |
| `-s3-endpoint` | — | S3 endpoint URL (leave empty for AWS) |
| `-s3-region` | — | S3 region |
| `-s3-access-key` | — | S3 access key |
| `-s3-secret-key` | — | S3 secret key |
| `-s3-bucket-name` | — | Target bucket |
| `-s3-key-pattern` | `$fileName` | Key pattern for uploaded objects |
| `-s3-include-files` | — | Glob pattern of files to include |
| `-s3-exclude-files` | — | Glob pattern of files to exclude |
| `-s3-keep-files` | `false` | Keep local files after uploading |
| `-s3-reduced-redundancy` | `false` | Use reduced redundancy storage class |
| `-s3-force-path-style` | `false` | Force path-style addressing (needed by some S3 clones) |

## Examples

Running the binary directly:

```bash
./websummoner -conf ~/.websummoner/websummoner/browsers.json -limit 10
```

Passing flags to the containerized version — everything after the image name
reaches `websummoner`:

```bash
docker run -d --name websummoner \
  -p 4444:4444 \
  -v ~/.websummoner/websummoner/:/etc/websummoner/:ro \
  -v /var/run/docker.sock:/var/run/docker.sock \
  websummoner/websummoner:latest-release \
  -conf /etc/websummoner/browsers.json -limit 10 -video-output-dir /opt/websummoner/video/
```
