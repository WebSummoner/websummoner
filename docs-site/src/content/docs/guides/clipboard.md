---
title: Clipboard
description: Read and write the clipboard of a running browser session.
---

To verify copy-paste behavior of your application, WebSummoner exposes the
clipboard of a running session over HTTP. The clipboard is accessible only
while the session is running.

With a session ID `f2bcd32b-d932-4cdc-a639-687ab8e4f840`:

**Read:**

```bash
curl http://websummoner-host.example.com:4444/clipboard/f2bcd32b-d932-4cdc-a639-687ab8e4f840
# some-clipboard-value
```

**Write:**

```bash
curl -X POST --data 'some-clipboard-value' \
    http://websummoner-host.example.com:4444/clipboard/f2bcd32b-d932-4cdc-a639-687ab8e4f840
```

A clipboard service listens inside the standard WebSummoner browser images —
no extra setup is needed.
