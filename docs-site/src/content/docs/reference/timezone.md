---
title: Timezone
description: Set the WebSummoner host timezone and per-session timezones.
---

## Host (container) timezone

The WebSummoner container defaults to **UTC**. Set the `TZ` environment
variable to change it:

```bash
docker run -d --name websummoner \
  -p 4444:4444 \
  -e TZ=Europe/Moscow \
  -v /etc/websummoner:/etc/websummoner:ro \
  -v /var/run/docker.sock:/var/run/docker.sock \
  websummoner/websummoner:latest-release
```

## Per-session timezone

Browser containers inherit the WebSummoner timezone by default. Tests that
need a specific zone can request one per session with the `timeZone`
capability — see the
[capabilities reference](/reference/capabilities/#per-session-time-zone-timezone).
