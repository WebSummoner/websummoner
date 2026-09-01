---
title: File download
description: Trigger downloads in the browser and pull the files out of the container.
---

Two separate problems: making the browser download without a "Save as"
dialog, and then getting the file out of the container. WebSummoner helps
with both.

## Suppressing the download dialog

**Chrome (Java)**

```java
ChromeOptions chromeOptions = new ChromeOptions();
chromeOptions.setExperimentalOption("prefs", Map.of(
    "profile.default_content_settings.popups", 0,
    "download.default_directory", "/home/selenium/Downloads",
    "download.prompt_for_download", false,
    "download.directory_upgrade", true,
    "safebrowsing.enabled", false,
    "plugins.always_open_pdf_externally", true
));
WebDriver driver = new RemoteWebDriver(
    new URL("http://localhost:4444/wd/hub"), chromeOptions);
driver.navigate().to("http://example.com/myfile.odt");
```

**Firefox (Java)**

```java
FirefoxOptions firefoxOptions = new FirefoxOptions();
firefoxOptions.setCapability("moz:firefoxOptions", Map.of(
    "prefs", Map.of(
        "browser.helperApps.neverAsk.saveToDisk", "application/octet-stream"
    )
));
WebDriver driver = new RemoteWebDriver(
    new URL("http://localhost:4444/wd/hub"), firefoxOptions);
driver.navigate().to("http://example.com/myfile.odt");
```

The usual in-container download directory is `~/Downloads`.

## Retrieving downloaded files

:::note
Works only when browsers run in containers, and only while the session is
still running.
:::

WebSummoner's `/download` API saves you from fiddling with container volumes.
With a running session `f2bcd32b-d932-4cdc-a639-687ab8e4f840`:

```text
GET    http://websummoner-host.example.com:4444/download/f2bcd32b-.../myfile.txt
DELETE http://websummoner-host.example.com:4444/download/f2bcd32b-.../myfile.txt
```

This relies on a small HTTP file server listening on port `8080` inside the
browser container — the standard WebSummoner images include it.
