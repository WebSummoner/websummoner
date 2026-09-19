---
title: HAR capture
description: Record a session's network traffic as a HAR 1.2 file, without a proxy in front of the browser.
---

WebSummoner can record everything a session requests and write it as a
[HAR 1.2](http://www.softwareishard.com/blog/har-12-spec/) file, readable by
browser devtools, Charles, Fiddler and most reporting tools.

Start the hub with a directory to write to:

```bash
websummoner -har-output-dir /opt/websummoner/har
```

then ask for it per session:

```json
{
  "capabilities": {
    "alwaysMatch": {
      "browserName": "chrome",
      "websummoner:options": {
        "enableHAR": true,
        "harName": "checkout-flow.har"
      }
    }
  }
}
```

`harName` is optional; without it the file is named after the session id.

The file is written when the session ends, and served from the hub:

| Request | Result |
| --- | --- |
| `GET /har/<name>.har` | The file |
| `GET /har/?json` | A JSON listing of every HAR file |
| `DELETE /har/<name>.har` | Removes it |

## How it works

The hub subscribes to the browser's own CDP Network domain for the life of the
session — there is no proxy in front of the browser, so HTTPS needs no
certificate and nothing about the page's network behaviour changes.

This makes HAR capture **Chromium-only**: chrome, MicrosoftEdge, opera, brave
and yandex. Firefox and WebKit expose no CDP endpoint, and requesting
`enableHAR` on them records nothing rather than failing the session.

Each entry carries the request method, URL and headers, the response status,
headers, MIME type and encoded size, and a total time derived from the CDP
timestamps. Response bodies are not stored, which keeps a long session's HAR
to a size worth keeping.
