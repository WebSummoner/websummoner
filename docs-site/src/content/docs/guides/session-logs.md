---
title: Session logs
description: Save a per-session log file, read its structured entries, and manage stored files.
---

WebSummoner can save a log file for every session — a detailed trace of what
happened while creating and running it. Files are named `<session-id>.log` by
default; change that per session with the `logName` capability (a plain file
name — no folders).

## Enabling

Run WebSummoner with `-log-output-dir`:

```bash
./websummoner -log-output-dir /path/to/some/dir
```

In Docker, mount the directory too:

```bash
docker run -d --name websummoner \
  -p 4444:4444 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $PWD/config/:/etc/websummoner/:ro \
  -v $PWD/logs/:/opt/websummoner/logs/ \
  websummoner/websummoner:latest-release -log-output-dir /opt/websummoner/logs
```

Only sessions with the `enableLog` capability are recorded — unless you start
with `-save-all-logs`, which records every session regardless.

## Downloading and deleting

Available after the session finishes (the temporary file is renamed to
`<session-id>.log` at session close):

```text
http://websummoner-host.example.com:4444/logs/<filename>.log   # direct link
http://websummoner-host.example.com:4444/logs/                  # list all
```

```bash
curl -X DELETE http://websummoner-host.example.com:4444/logs/<filename>.log
```

There is no automatic retention — clean up with a scheduled job, like in the
[video guide](/guides/video-recording/#deleting-videos).

## Reading a log file

A typical log:

```text
2017/11/01 19:12:38 [-] [NEW_REQUEST]
2017/11/01 19:12:38 [-] [NEW_REQUEST_ACCEPTED]
2017/11/01 19:12:38 [41301] [LOCATING_SERVICE] [firefox-154.0]
2017/11/01 19:12:38 [41301] [USING_DOCKER] [firefox-154.0]
2017/11/01 19:12:39 [41301] [CREATING_CONTAINER] [websummoner/firefox:154.0]
2017/11/01 19:12:40 [41301] [CONTAINER_STARTED] [websummoner/firefox:154.0] [19760edf...] [896ms]
2017/11/01 19:12:40 [41301] [SERVICE_STARTED] [websummoner/firefox:154.0] [19760edf...] [605ms]
2017/11/01 19:12:40 [41301] [PROXY_TO] [websummoner/firefox:154.0] [19760edf...] [http://172.17.0.3:4444/]
2017/11/01 19:12:42 [41301] [SESSION_CREATED] [test-quota] [345bb886-...] [http://172.17.0.3:4444/] [1] [4.15s]
2017/11/01 19:14:30 [41301] [SESSION_DELETED] [345bb886-...]
```

Each entry has bracketed fields:

| Field | Example | Meaning |
| --- | --- | --- |
| Time | `19:12:42` | When the entry was written |
| Counter | `[41301]` | Request counter — the session ID does not exist yet during attempts, so this groups all lines of one new-session request |
| Status | `[SESSION_CREATED]` | What happened — see table below |
| Browser | `[firefox-154.0]` | Name and version (new-session requests only) |
| Attempt | `[1]` | Attempt number; for `SESSION_CREATED` — total attempts used |
| Session ID | `[345bb886-...]` | Unique per browser session |
| Duration | `[4.15s]` | Time spent |

### Status reference

| Status | Meaning |
| --- | --- |
| `NEW_REQUEST` / `NEW_REQUEST_ACCEPTED` | Session request received / accepted |
| `LOCATING_SERVICE` | Matching the request against `browsers.json` |
| `USING_DOCKER` / `USING_DRIVER` | Container mode or standalone driver chosen |
| `CREATING_CONTAINER` / `STARTING_CONTAINER` / `CONTAINER_STARTED` | Container lifecycle |
| `ALLOCATING_PORT` / `ALLOCATED_PORT` | Free-port allocation for driver processes |
| `STARTING_PROCESS` / `PROCESS_STARTED` | Driver process lifecycle (drivers mode) |
| `SERVICE_STARTED` | Browser service is up |
| `PROXY_TO` | Forwarding WebDriver traffic to the service |
| `SESSION_ATTEMPTED` / `SESSION_CREATED` / `SESSION_FAILED` | New-session outcomes |
| `SESSION_DELETED` | Session removed |
| `VIDEO_RECORDING` / `VIDEO_SAVED` | Video recorder attached / file saved |
| `LOG_SAVED` | Log file saved |
| `FILE_CREATED` | Uploaded file stored |

When a session misbehaves, the fastest path is: find its `SESSION_CREATED`
line, take the session ID, and read upwards — the entry just before the first
failure tells you whether the problem is image resolution, container startup,
or the WebDriver handshake.
