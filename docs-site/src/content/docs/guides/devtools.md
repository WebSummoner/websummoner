---
title: DevTools protocol
description: Drive Chrome DevTools (CDP) for a Selenium session — screenshots, metrics, tracing, Puppeteer.
---

For every running session WebSummoner proxies the
[Chrome DevTools protocol](https://chromedevtools.github.io/devtools-protocol/)
to the browser container. Point any CDP client at:

```text
<ws-or-http>://websummoner.example.com:4444/devtools/<session-id>/<method>
```

:::note
CDP is a Chromium protocol, so this works for the Chromium-based images —
Chrome, Edge, Opera, Yandex and Brave. Firefox and WebKit have no CDP endpoint,
and WebSummoner deliberately does not advertise one for them: a client that saw
`se:cdp` there would fail on its first WebSocket handshake.
:::

:::caution
CDP is on its way out. Firefox disabled it by default in Firefox 129, Selenium
removed its Firefox CDP support in 4.29, and the cross-browser replacement is
[WebDriver BiDi](https://w3c.github.io/webdriver-bidi/). WebSummoner does not
proxy BiDi yet — if your tests need bidirectional features on Firefox or WebKit,
that is not available through the grid today. Prefer BiDi in new test code where
your client supports it directly.
:::

## Supported methods

| Method | Protocol | Meaning |
| --- | --- | --- |
| `/browser` | WebSocket | Browser-level DevTools websocket |
| `/` | WebSocket | Alias for `/browser` |
| `/page` | WebSocket | Current page (target) websocket — ideal when a single tab is open |
| `/page/<target-id>` | WebSocket | Websocket for a specific tab |
| `/json/protocol` | HTTP | Supported protocol methods as JSON (used by some client libraries) |

For example, the current-page websocket is:

```text
ws://websummoner.example.com:4444/devtools/<session-id>/page
```

## Example: Puppeteer over a WebdriverIO session

Because the Selenium session and the CDP connection share the same browser,
you can mix high-level Selenium actions with Puppeteer's low-level access:

```javascript
const { remote } = require('webdriverio');
const puppeteer = require('puppeteer-core');
const host = 'websummoner.example.com';

(async () => {
    const browser = await remote({
        hostname: host,
        capabilities: {
            browserName: 'chrome',
            browserVersion: '152.0',
        },
    });

    const devtools = await puppeteer.connect({
        browserWSEndpoint: `ws://${host}:4444/devtools/${browser.sessionId}`,
    });

    const page = await devtools.newPage();
    await page.goto('https://example.com');
    await page.screenshot({ path: 'screenshot.png' });
    console.log(await page.title());

    await devtools.close();
    await browser.deleteSession();
})().catch((e) => console.error(e));
```
