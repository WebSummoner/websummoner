---
title: Comparing WebSummoner
description: A source-checked comparison with Selenoid, Moon, Selenium Grid, Playwright and Puppeteer — including when to choose them instead.
---

# Comparing WebSummoner

Every competitor claim on this page is taken from the product's **official
documentation** (sources linked at the bottom, last verified August 30,
2026). WebSummoner column reflects this documentation set. If any of this
goes stale, please open an issue — an outdated comparison is worse than none.

Two of the tools below are the same *kind* of thing as WebSummoner (session
infrastructure); two are a different kind (test toolchains your tests are
written with).

## vs. other session infrastructure

| | WebSummoner | Selenoid | Moon | Selenium Grid 4 |
| --- | --- | --- | --- | --- |
| Runs on | Docker host or VM | Docker host or VM | **Kubernetes/OpenShift only** (Helm chart) | Standalone, hub-node or fully distributed mode |
| Maintenance | Actively maintained by [RIADVICE](https://riadvice.com) | **Archived** by its owner on 17 December 2024 — read-only | Commercially maintained | Selenium project + official Docker images |
| Runtime footprint | ~10 MB Go binary, no JVM | Same | Stateless pods, replicable across datacenters | **Java 11+**; docs estimate ~1 GB RAM per session |
| Protocols served | WebDriver (W3C) | WebDriver | **WebDriver + Playwright + Cypress + Puppeteer/CDP** | WebDriver |
| Client languages | Any Selenium client | Same | All of the above | Any Selenium client |
| Session limits & queue | `-limit`, built-in queue, retries, GGR for clusters | Same | Unlimited namespaces, per-team quota resources, resource requests/limits | FIFO session queue (default 300 s timeout / 5 s retry); CPU-based slots (one for Safari) |
| Prometheus metrics | `/metrics` endpoint — sessions, queue, per-browser gauges | No | Via Moon UI / Prometheus | No built-in metrics endpoint |
| Video recording | H.264 **with audio** (`enableVideo`/`enableAudio`) | Video only, no audio | H.264 with audio (`enableAudio`, since 2.7.5) + **HAR capture** | ffmpeg **sidecar container per browser container**, `se:recordVideo`, failure-only retention, RCLONE upload; **not supported for headless** |
| Live screen | VNC over WebSocket + UI | Same | VNC + UI (view-only by default) | noVNC web view (port 7900) |
| Clipboard / files | Clipboard API, download & upload APIs | Same | Clipboard (text + PNG), downloads REST API, `context` archive capability | Downloads/uploads via Selenium's file detector; no clipboard API |
| Browsers | [Maintained images](/reference/browser-images/) | Same, older set | Chrome, Firefox, Edge, Opera, Safari (+ Playwright/Cypress sets); real mobile not possible, IE via external hosts | Chrome, Chromium, Firefox, Edge + CfT; multi-arch amd64/arm64 |
| Container orchestration | Not needed (Docker only); **not for Kubernetes** | Same | Kubernetes-native, multiple namespaces | Official Helm chart + KEDA autoscaling + `SE_DRAIN_AFTER_SESSION_COUNT` |
| License | Apache-2.0 | Apache-2.0 | **Commercial** — free for up to 4 parallel sessions, paid license beyond | Apache-2.0 |

### When to choose the others

- **Moon** — if you are on Kubernetes, need per-team quotas, or want to serve
  Playwright/Cypress/Puppeteer sessions from the same infrastructure. Note
  the license: free usage is capped at 4 parallel sessions.
- **Selenium Grid** — if you want the reference implementation, its
  distributed mode, or the KEDA-based autoscaling; the price is a JVM
  footprint and, for video, a 1:1 sidecar container per browser container.
  Its Dynamic Grid (Node configs mapping capabilities to Docker images) is
  conceptually close to WebSummoner's `browsers.json`.
- **Selenoid** — no reason anymore; the repository was archived in December 2024 and is read-only.
  [Migration](/migrating-from-selenoid/) is intentionally boring.
- **WebSummoner** — Docker hosts or VMs, low operational weight, batteries
  included (video with audio, VNC, clipboard/files, GGR clustering), Apache-2.0.

## vs. test toolchains

Selenide, Playwright and Puppeteer are what your tests are *written with*;
WebSummoner is what serves sessions *to* WebDriver clients. They come up in the
same conversations, so here is the honest overlap. Note the last row: only the
tools that speak WebDriver can use a grid at all.

| | WebSummoner + Selenium clients | Selenide | Playwright | Puppeteer |
| --- | --- | --- | --- | --- |
| Category | Session infrastructure | Concise API **on top of** Selenium WebDriver — adds implicit waiting, assertions and lifecycle | "End-to-end test framework" — bundles runner, assertions, isolation, parallelization | "JavaScript library to control Chrome or Firefox" |
| Languages | Java, Python, C#, Ruby, JS/TS, … | Java (JVM) | Node.js, Python, Java, .NET | JavaScript/TypeScript |
| Browsers | Pinned Docker images server-side | Whatever the grid or local driver provides | Bundled Chromium, WebKit, Firefox (patched builds) + branded Chrome/Edge channels + mobile emulation | Chrome (bundled by default) and Firefox |
| Protocol | WebDriver wire protocol | WebDriver — it *is* a Selenium client | Own client + patched browser builds — not the WebDriver protocol | **CDP or WebDriver BiDi** |
| Parallelism | The hub (`-limit`, queue) + GGR | Your JUnit/TestNG runner, plus the hub | Built-in parallel runs and sharding | Not provided by the library |
| Recording & debugging | Video (with audio), VNC, logs, DevTools proxy | Automatic screenshots on failure; video and VNC come from the grid | Trace viewer, videos, screenshots, UI mode with time-travel debugging | Not covered by its docs landing page |
| Can use WebSummoner? | — | **Yes, natively** — point `Configuration.remote` at the hub | **No** — it does not speak WebDriver | Partially — Puppeteer can attach to a WebDriver session through our [DevTools endpoint](/guides/devtools/) |

**Choose Selenide** for JVM projects that want terse, readable tests without
leaving the WebDriver standard. It is a client library, not infrastructure, so it
complements WebSummoner rather than competing with it — point it at the hub and
every capability on this site still applies:

```java
Configuration.remote = "http://websummoner.example.com:4444/wd/hub";
Configuration.browser = "chrome";
```

One caveat worth knowing: Selenide surfaces the same browser limitations
everything else does. Its issue tracker carries the Safari proxy case
([selenide#1575](https://github.com/selenide/selenide/issues/1575), closed as
*not a bug*) — SafariDriver rejects the `proxy` capability, which WebSummoner
works around for the WebKit image (see
[Proxies on WebKit](/reference/browser-images/#proxies-on-webkit)).

**Choose Playwright** for greenfield JS/TS (or Java/Python/.NET) work when
you control the stack: the tracing and debugging tooling is excellent and
there is no grid to operate. The trade-off is portability — patched browser
builds and a non-WebDriver protocol tie you to its ecosystem.

**Choose Puppeteer** for lightweight CDP-level automation of Chrome/Firefox
in JavaScript.

**Keep Selenium + WebSummoner** when you need the W3C-standard wire
protocol, polyglot clients, or server-side control over pinned browser
environments.

## Commercial clouds

BrowserStack, Sauce Labs, LambdaTest and similar services replace
infrastructure with a per-minute bill: no servers, huge device matrices
(including real mobile). They complement a self-hosted hub — WebSummoner for
CI, a cloud for exotic coverage — rather than competing with it.

## Sources

Last verified **August 30, 2026**:

- Moon — [official documentation (latest, dated 2026-04-01)](https://aerokube.com/moon/latest/):
  multi-protocol support (Selenium, Playwright, Cypress, Puppeteer), `enableAudio`
  since 2.7.5, `enableHAR` since 2.7.2, namespaces/quotas, statelessness,
  free-tier limit of 4 parallel sessions, commercial model.
- Selenium Grid — [Components](https://www.selenium.dev/documentation/grid/components/)
  and [Getting Started](https://www.selenium.dev/documentation/grid/getting_started/):
  architecture, FIFO queue defaults, CPU-based slots, Java 11+ requirement,
  ~1 GB RAM per session estimate.
- Selenium Docker packaging — [SeleniumHQ/docker-selenium](https://github.com/SeleniumHQ/docker-selenium):
  video sidecar (`selenium/video`, `se:recordVideo`, failure-only retention,
  headless limitation), noVNC, session-queue env vars, Helm chart and KEDA.
- Playwright — [Introduction](https://playwright.dev/docs/intro) and
  [Browsers](https://playwright.dev/docs/browsers): framework scope,
  languages, bundled patched browsers, branded channels, parallelization.
- Puppeteer — [official docs](https://pptr.dev/): library scope, Chrome +
  Firefox, "DevTools Protocol or WebDriver BiDi", bundled Chrome download.
