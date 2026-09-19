---
title: Session queue
description: How WebSummoner decides whether a new session waits for a slot or is refused, and how to tune it.
---

When all `-limit` slots are busy, a new request queues. How long it may wait
before giving up with `429` is decided by a single number: **how long since a
request was admitted quickly**. Call it `idle`.

- A *fast* admission resets `idle` to zero.
- Nothing else resets it: not a slow admission, not a refusal, not the queue
  emptying.

| `idle` | State | Budget given to a newly queued request |
| --- | --- | --- |
| ≤ `-queue-timeout` | Keeping up | `-queue-timeout` |
| > `-queue-timeout` | Saturated | `-queue-congestion-timeout` |

That is the entire rule. You normally set one flag:

```bash
websummoner -limit 2 -queue-timeout 5m
```

and the three numbers it works with are derived from it:

| Number | Derived as | With `-queue-timeout 5m` |
| --- | --- | --- |
| How fast an admission must be to count | `-queue-timeout` / 20, minimum 1s | **15s** |
| Budget while keeping up | `-queue-timeout` itself | **5m** |
| Budget while saturated | `-queue-congestion-timeout`, which defaults to `-queue-timeout` / 10, minimum 1s | **30s** |

Only the last one has its own flag; set `-queue-congestion-timeout` to override
it, and an explicit value is used as given rather than floored. The other two
always follow `-queue-timeout`, so there is one knob to tune in practice.

Leaving `-queue-timeout` unset disables all of this: requests wait indefinitely
until the client gives up, as Selenoid did.

## What it looks like

**Grid keeping up.** Sessions finish, so admissions stay fast and keep resetting
`idle`:

| Time | Event | `idle` after | Budget given |
| --- | --- | --- | --- |
| 10:00:00 | A and B admitted, no wait | 0 | — |
| 10:00:04 | C queues | 4s | 5m |
| 10:00:09 | A ends, C admitted after 5s — fast | 0 | — |
| 10:02:00 | D queues | 1m51s | 5m |

`idle` never approaches 5m, so every request gets the lenient budget. A burst is
absorbed without the grid ever changing state.

**Grid jammed.** A broken browser image hangs on startup, so A and B never
finish and nothing is ever admitted again. With no fast admissions, `idle` only
grows:

| Time | Event | `idle` | Budget given |
| --- | --- | --- | --- |
| 10:00:00 | A and B admitted, then hang | 0 | — |
| 10:00:10 | C queues | 10s | 5m |
| 10:05:10 | C refused, its 5m is up | 5m10s | — |
| 10:05:11 | D queues | 5m11s | 30s — saturated |
| 10:05:41 | D refused, its 30s is up | 5m41s | — |

C still waited the full five minutes, and that is correct: until `idle` crosses
`-queue-timeout`, a jammed grid is indistinguishable from a merely busy one.
From D onwards every request fails in 30s, so CI reports a broken grid in half a
minute instead of tying up a build slot for five.

This is [CoDel](https://queue.acm.org/detail.cfm?id=2209336), the
controlled-delay algorithm from network queue management, applied to session
slots rather than packets. `-queue-timeout` is CoDel's *interval* and the fast
threshold is its *target*, kept at the same 5% of the interval the original
uses.

## Why wait time and not queue depth

The decision is driven by how long ago something was served, not by how many
requests are queued right now. Depth is misleading here: on a saturated grid
every waiter eventually times out and leaves, so the queue keeps emptying and
reads as idle exactly when things are worst. Time since the last fast admission
has no such blind spot, which is why CoDel is built on it.

## Every outcome

| Situation | Result |
| --- | --- |
| `-disable-queue` is set | Immediate error, no waiting |
| `X-WebSummoner-No-Wait` (or legacy `X-Selenoid-No-Wait`) sent | Immediate `429 Too Many Requests` |
| The wait budget elapses | `429 Too Many Requests` |
| The client disconnects first | Nothing — the slot is released, no response needed |
| A slot frees first | The session starts normally |

## Retry-After

Both `429` responses carry a `Retry-After` header, randomised over `[1,
-queue-timeout]` seconds. The randomness matters: a fixed value puts every
rejected client on the same second, so they all come back together and jam the
grid again. Honour the header as sent rather than rounding it.

## Telling the two refusals apart

Both are `429`. The hub logs them as `[QUEUE_TIMEOUT]` and `[QUEUE_SHED]`, and
[`/metrics`](/guides/usage-statistics/#available-metrics) counts them
separately:

| Metric | Means |
| --- | --- |
| `websummoner_queue_congested` | `1` while shedding — alert on this, not on rejection counts |
| `websummoner_queue_rejected_total{reason="no_wait"}` | Client opted out of waiting, usually ggr |
| `websummoner_queue_rejected_total{reason="timeout"}` | Slow grid, queue otherwise healthy — consider raising `-limit` |
| `websummoner_queue_rejected_total{reason="shed"}` | Saturated grid — needs capacity, not patience |
