---
title: Uploading to S3
description: Ship recorded video and log files to S3-compatible storage automatically.
---

WebSummoner can upload recorded video and log files to any S3-compatible
storage as sessions finish.

:::important
Only available in binaries built with the `s3` build tag — the official
release binaries and the standard Docker image include it. Building from
source: `go build -tags s3 .`
:::

## Enabling

Pass your S3 credentials and bucket:

```bash
./websummoner \
  -s3-endpoint https://s3.us-east-2.amazonaws.com \
  -s3-region us-east-2 \
  -s3-bucket-name my-bucket \
  -s3-access-key <your-access-key> \
  -s3-secret-key <your-secret-key>
```

If you omit `-s3-access-key` and `-s3-secret-key`, the standard AWS fallbacks
apply (environment variables, shared credentials file).

For S3 clones like MinIO, add `-s3-force-path-style` and point
`-s3-endpoint` at your installation.

## Key patterns

By default the file name is preserved (`/<session-id>.log`,
`/myCustomVideo.mp4`). Set `-s3-key-pattern` to reorganize, using these
placeholders:

| Placeholder | Replaced by |
| --- | --- |
| `$browserName` | `browserName` capability |
| `$browserVersion` | browser version capability |
| `$date` | current date, e.g. `2018-11-01` |
| `$fileName` | source file name |
| `$fileExtension` | source file extension |
| `$fileType` | `log` or `video` |
| `$platformName` | platform capability |
| `$quota` | quota name (usually from GGR) |
| `$sessionId` | Selenium session ID |

Example — with `-s3-key-pattern "$browserName/$sessionId/log.txt"` a log lands
at `firefox/0ee0b48b-.../log.txt`. The pattern can also be overridden per
session with the `s3KeyPattern`
[capability](/reference/capabilities/#s3-key-pattern-s3keypattern).

## Filtering uploads

To upload only some files, use globs:

```bash
-s3-include-files '*.mp4'   # only video
-s3-exclude-files '*.log'   # everything except logs
```

By default local files are deleted after a successful upload — keep them with
`-s3-keep-files`.
