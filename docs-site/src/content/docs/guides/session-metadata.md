---
title: Session metadata
description: Save a JSON metadata file next to every session log for post-processing.
---

WebSummoner can save session metadata — ID, capabilities, start and finish
times — as a separate JSON file. The main use case is post-processing session
artifacts, for example feeding a MapReduce pipeline that correlates test
results with recorded video and logs.

:::important
Only available in binaries built with the `metadata` build tag — the official
release binaries include it. Building from source: `go build -tags metadata .`
:::

No extra configuration: when the feature is compiled in **and**
`-log-output-dir` is set, WebSummoner writes
`<log-output-dir>/<session-id>.json` for every session:

```json
{
  "id": "62a4d82d-edf6-43d5-886f-895b77ff23b7",
  "capabilities": {
    "browserName": "chrome",
    "version": "152.0",
    "name": "MyCoolTest",
    "screenResolution": "1920x1080x24"
  },
  "started": "2026-08-30T16:23:12.440916+03:00",
  "finished": "2026-08-30T16:23:12.480928+03:00"
}
```
