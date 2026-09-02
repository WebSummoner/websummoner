---
title: Usage statistics
description: The /status endpoint explained — limits, queue, pending and per-browser usage.
---

WebSummoner keeps live usage statistics, exposed at:

```bash
curl http://localhost:4444/status
```

```json
{
  "total": 80,
  "used": 10,
  "queued": 0,
  "pending": 1,
  "browsers": {
    "firefox": {
      "155.0": {
        "user1": {
          "count": 1,
          "sessions": [
            {
              "id": "a7a2b801-21db-4dae-a99b-4cbc0b81de96",
              "vnc": false,
              "screen": "1920x1080x24"
            }
          ]
        },
        "user2": { "count": 6, "sessions": [] }
      }
    }
  }
}
```

Users are taken from basic HTTP authentication headers.

## What the numbers mean

A session request goes through a small state machine:

![Queued to Pending to Attempted, then either Created or Failed](/img/lifecycle.png)

1. **Request arrives.** The `-limit` flag caps simultaneous sessions — this is
   `total`.
2. **Queue.** When all slots are taken, new requests block in `queued`. A
   request carrying the `X-WebSummoner-No-Wait` header skips the queue
   entirely: if no slot is free, WebSummoner replies immediately with
   `429 Too Many Requests` instead of making the client wait.
3. **Pending.** A freed slot starts a container or driver process; requests
   during startup count as `pending`. WebSummoner waits for the service port
   to answer before proceeding.
4. **Created / failed.** WebSummoner performs the WebDriver new-session call
   itself; success adds to `used`, failure returns an error to the client.
5. **Proxying.** From then on, all session traffic is proxied to the same
   container or driver until the session is deleted.

So a healthy idle hub shows `used` close to your real load and `queued`/`pending`
near zero; a persistently non-zero `queued` means you need a higher `-limit`
(or more hub instances behind [GGR](https://github.com/WebSummoner/ggr)).

## Prometheus metrics

WebSummoner exposes a `/metrics` endpoint in Prometheus text format — no
client library needed, point your scraper at it:

```yaml
scrape_configs:
  - job_name: websummoner
    static_configs:
      - targets: ["websummoner-host:4444"]
```

### Available metrics

| Metric | Type | Meaning |
| --- | --- | --- |
| `websummoner_sessions_active` | gauge | Currently running browser sessions |
| `websummoner_sessions_limit` | gauge | Maximum simultaneous sessions (`-limit` flag) |
| `websummoner_queue_depth` | gauge | Requests waiting in the queue |
| `websummoner_queue_pending` | gauge | Requests being processed |
| `websummoner_browser_sessions{browser="chrome:152.0"}` | gauge | Sessions per browser/version |
| `websummoner_sessions_created_total` | counter | Sessions successfully created |
| `websummoner_sessions_failed_total` | counter | Sessions that failed to start |
| `websummoner_sessions_timed_out_total` | counter | Sessions closed by idle timeout |
| `websummoner_sessions_deleted_total` | counter | Sessions deleted by client |
| `websummoner_video_sessions_total` | counter | Sessions with video recording |
| `websummoner_vnc_sessions_total` | counter | Sessions with VNC enabled |
| `websummoner_audio_sessions_total` | counter | Sessions with audio recording |

Example output:

```text
# HELP websummoner_sessions_active Currently running browser sessions
# TYPE websummoner_sessions_active gauge
websummoner_sessions_active 3
websummoner_sessions_limit 5
websummoner_queue_depth 0
```

### Suggested alerts

```yaml
- alert: WebSummonerQueueBacklog
  expr: websummoner_queue_depth > 10
  for: 5m

- alert: WebSummonerHighFailureRate
  expr: rate(websummoner_sessions_failed_total[5m]) > 0.1
  for: 10m

- alert: WebSummonerAtCapacity
  expr: websummoner_sessions_active / websummoner_sessions_limit > 0.9
  for: 15m
```

## Health checks for orchestrators

Kubernetes liveness/readiness probes and load balancer health checks often
send `HEAD` requests (no body). WebSummoner supports both methods on its
health endpoints:

- `HEAD /ping` → `200 OK` (always, when the process is alive)
- `GET /ping` → JSON with uptime, last reload time and request count
- `HEAD /status` → `200 OK` when ready (slots available), `503 Service Unavailable` when full
- `GET /status` → the full usage JSON shown above

Example Kubernetes probe:

```yaml
livenessProbe:
  httpGet:
    path: /ping
    port: 4444
readinessProbe:
  httpGet:
    path: /status
    port: 4444
```

## Shipping statistics elsewhere

Any collector that reads JSON over HTTP works — for example
[Telegraf](https://github.com/influxdata/telegraf) forwarding to
[Graphite](https://github.com/graphite-project):

1. Generate a config:

   ```bash
   mkdir -p /etc/telegraf
   docker run --rm telegraf:alpine \
       --input-filter httpjson \
       --output-filter graphite config > /etc/telegraf/telegraf.conf
   ```

2. Point it at the status endpoint:

   ```ini
   [agent]
   interval = "10s"

   [[outputs.graphite]]
   servers = ["my-graphite-host.example.com:2024"]
   prefix = "one_min"
   template = "host.measurement.field"

   [[inputs.httpjson]]
   name = "websummoner"
   servers = ["http://localhost:4444/status"]
   ```

3. Run it:

   ```bash
   docker run --net host -d --name telegraf \
     -v /etc/telegraf:/etc \
     telegraf:alpine --config /etc/telegraf.conf
   ```
