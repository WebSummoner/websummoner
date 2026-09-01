---
title: Docker Compose
description: Run WebSummoner with Compose — bridge network or a custom network.
---

Both examples assume this layout:

```bash
mkdir -p /path/to/config /path/to/config/logs /path/to/config/video
# put browsers.json into /path/to/config
```

Both options follow the official
[Docker Compose documentation](https://docs.docker.com/compose/).

## Option 1 — default Docker network

All services use `bridge` networking:


```yaml
services:
  websummoner:
    network_mode: bridge
    image: websummoner/websummoner:latest-release
    volumes:
      - "/path/to/config:/etc/websummoner"
      - "/var/run/docker.sock:/var/run/docker.sock"
      - "/path/to/config/video:/opt/websummoner/video"
      - "/path/to/config/logs:/opt/websummoner/logs"
    environment:
      - OVERRIDE_VIDEO_OUTPUT_DIR=/path/to/config/video
    command:
      - -conf
      - /etc/websummoner/browsers.json
      - -video-output-dir
      - /opt/websummoner/video
      - -log-output-dir
      - /opt/websummoner/logs
    ports:
      - "4444:4444"
```

## Option 2 — custom Docker network

When your application containers live in their own network, create it first
([Docker networking guide](https://docs.docker.com/engine/network/tutorials/standalone/)):

```bash
docker network create websummoner
```

Then attach WebSummoner to it **and** pass `-container-network` so browser
containers join the same network:

```yaml
networks:
  websummoner:
    external: true

services:
  websummoner:
    networks:
      websummoner:
    image: websummoner/websummoner:latest-release
    volumes:
      - "/path/to/config:/etc/websummoner"
      - "/var/run/docker.sock:/var/run/docker.sock"
      - "/path/to/config/video:/opt/websummoner/video"
      - "/path/to/config/logs:/opt/websummoner/logs"
    environment:
      - OVERRIDE_VIDEO_OUTPUT_DIR=/path/to/config/video
    command:
      - -conf
      - /etc/websummoner/browsers.json
      - -video-output-dir
      - /opt/websummoner/video
      - -log-output-dir
      - /opt/websummoner/logs
      - -container-network
      - websummoner
    ports:
      - "4444:4444"
```

:::tip
The `OVERRIDE_VIDEO_OUTPUT_DIR` value must equal the **host** path of the
video volume — see [Video recording](/guides/video-recording/).
:::
