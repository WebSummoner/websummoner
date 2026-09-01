---
title: Without Docker (drivers mode)
description: Use WebSummoner as a lightweight Selenium server for local browsers — IE on Windows, desktop Chrome and Firefox.
---

Browsers usually run in containers, but sometimes that is impossible — the
classic case is Internet Explorer on Windows, which cannot be containerized.
In **drivers mode** WebSummoner starts browser driver binaries as regular
processes and acts as a lightweight Selenium server replacement.

## Example: Internet Explorer on Windows

1. Download the [IEDriverServer](https://www.selenium.dev/downloads/) archive
   and unpack it (to `C:\` in this example).
2. Download the
   [WebSummoner binary](https://github.com/WebSummoner/websummoner/releases/latest).
3. Create `browsers.json` — note that `image` is a **command array**, and
   Windows backslashes must be escaped as `\\`:

   ```json
   {
     "internet explorer": {
       "default": "11",
       "versions": {
         "11": {
           "image": ["C:\\IEDriverServer.exe", "--log-level=DEBUG"]
         }
       }
     }
   }
   ```

4. Start WebSummoner without Docker:

   ```bash
   ./websummoner_windows_amd64.exe -conf ./browsers.json -disable-docker
   ```

5. Run tests against `http://localhost:4444/wd/hub` with:

   ```text
   browserName = internet explorer
   version     = 11
   ```

## Other browsers

Download the [ChromeDriver](https://chromedriver.chromium.org/) binary and
point the `image` array at it:

```json
{
  "chrome": {
    "default": "152",
    "versions": {
      "152": {
        "image": ["/usr/bin/chromedriver", "--port=4444"]
      }
    }
  }
}
```

## Driver logs

By default driver process output is discarded. Add `-capture-driver-logs` to
append every session's driver log to the main WebSummoner log.

:::note
[File upload](/guides/file-upload/#running-without-docker) needs
`-enable-file-upload` in drivers mode, because some drivers (geckodriver,
IEDriver) do not implement the `/file` endpoint themselves.
:::
