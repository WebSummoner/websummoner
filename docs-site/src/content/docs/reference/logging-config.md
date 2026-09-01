---
title: Container logging configuration
description: Route browser container logs to a centralized logging driver.
---

By default Docker writes container logs to the host filesystem — fine for
local development. In a big cluster you usually want logs in centralized
storage such as [Logstash](https://www.elastic.co/logstash) or
[Graylog](https://www.graylog.org/).

Docker supports this through
[logging drivers](https://docs.docker.com/engine/logging/), and WebSummoner
lets you set the driver globally for every browser container it starts, via
the `-log-conf` flag.

## File format

`config/container-logs.json`:

```json
{
  "Type": "syslog",
  "Config": {
    "syslog-address": "tcp://192.168.0.42:123",
    "syslog-facility": "daemon"
  }
}
```

- `Type` — any Docker logging driver: `syslog`, `journald`, `awslogs`, …
- `Config` — key-value options for that driver.

The example above is equivalent to running containers with:

```bash
--log-driver=syslog \
--log-opt syslog-address=tcp://192.168.0.42:123 \
--log-opt syslog-facility=daemon
```

## Using it

```bash
./websummoner -log-conf config/container-logs.json
```

:::note
This controls the **browser container** stdout/stderr logs. Per-session
WebDriver logs saved as files are a separate feature — see
[Session logs](/guides/session-logs/).
:::
