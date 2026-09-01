---
title: FAQ & troubleshooting
description: Common questions and fixes — Kubernetes, timeouts, video and VNC issues.
---

## Kubernetes

**Can WebSummoner run in Kubernetes?** Possible, but not recommended:

- WebSummoner talks directly to the
  [Docker API](https://docs.docker.com/engine/api/) and is designed for a
  workstation or VM with Docker installed; Kubernetes has a completely
  different API and may not even use Docker as its runtime.
- Even when it works, all browser containers start **on the same node** as
  the WebSummoner pod — invisible to Kubernetes scheduling, which can
  overload the node.
- Only **one** WebSummoner replica is possible: the session list lives in
  memory.

For Kubernetes-native Selenium, see the alternatives discussed in the
[comparison page](/reference/compare/).

## Logs and directories

**Where are WebSummoner logs?** On stdout. For the container:

```bash
docker logs -f websummoner
```

**Where are recorded videos stored?** When installed with `cm`:
`~/.websummoner/websummoner/video` (or
`C:\Users\<user>\.websummoner\websummoner\video` on Windows).

## Limits and timeouts

**What HTTP status do I get when the queue is full?**
If you send the `X-WebSummoner-No-Wait` header, WebSummoner replies with
**429 Too Many Requests** immediately. Without the header, the request
blocks in the queue until a slot is free (or the client disconnects).

**Limit overall browser consumption?** The `-limit` flag — total parallel
sessions, default 5. See [Recommended Docker settings](/reference/docker-settings/)
for sizing.

**Limit per-version consumption?** Not supported — by design, the only
sensible limit is overall consumption.

**Adjust timeouts?** The main one is `-timeout` (max time between WebDriver
requests before a session is closed). The subtle ones:
`-service-startup-timeout`, `-session-attempt-timeout`,
`-session-delete-timeout` — see the [CLI flags reference](/reference/cli-flags/).

## Resources

**How much do browser containers consume?** Depends on tests; start with
1 CPU + 1 GB per container and raise `-limit` while watching stability.

**Are there separate VNC images?** No. Every image embeds `x11vnc`, and the
server only runs when the session asks for it with `enableVNC` (or
`ENABLE_VNC=true` standalone). Idle cost when it does run is about 20 MB —
negligible next to the browser.

**VNC eats all CPU?** On RedHat-based distributions set
`LimitNOFILE=1048576` for `containerd.service`.

## Fixes for common errors

**`open config/browsers.json: no such file or directory`** — you overrode
the container command and dropped the config path. Add it back:

```bash
docker run <args> websummoner/websummoner:latest-release \
  -limit 10 -conf /etc/websummoner/browsers.json
```

**`create container: ... client version 1.36 is too new`** — pin the Docker
API version to your daemon (`docker version | grep API`):

```bash
DOCKER_API_VERSION=1.32 ./websummoner <args>          # binary
docker run -e DOCKER_API_VERSION=1.32 <args> websummoner/websummoner:latest-release
```

**Video files keep random names and `VIDEO_ERROR` in logs** — the recorder
container cannot rename the temp file. Check that:

1. `OVERRIDE_VIDEO_OUTPUT_DIR` points to the **host** directory where videos
   are stored.
2. When passing custom args to the container you also pass
   `-video-output-dir /opt/websummoner/video` and mount the host video
   directory at `/opt/websummoner/video`.

**VNC shows "Disconnected"** — the session lacks the `enableVNC: true`
capability.

**VNC shows a black screen with a cross** — the test called `driver.close()`
(closes the last tab) instead of `driver.quit()` (ends the session).

## Miscellaneous

**Can WebSummoner pull browser images automatically?** No — intentional.
Automatic pulls under load would make startup times and failures
unpredictable. Use `cm` or the `jq` snippet in
[Browsers configuration](/reference/browsers-config/).

**Does it work in a Docker macvlan network?** Yes — background on the
driver itself: [macvlan network driver](https://docs.docker.com/engine/network/drivers/macvlan/);
see [aerokube/selenoid#795](https://github.com/aerokube/selenoid/issues/795)
for working setups (the thread predates the fork).
