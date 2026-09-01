---
title: File upload
description: Upload files from the test machine into the browser session.
---

Most Selenium clients implement the WebDriver `/file` endpoint, so uploading
usually works out of the box — the client zips the local file and pushes it to
the hub, and the browser sees it on its own filesystem.

## Client snippets

**Java**

```java
WebElement input = driver.findElement(By.cssSelector("input[type='file']"));

// Make sure the element is visible
((JavascriptExecutor) driver).executeScript(
    "arguments[0].style.display = 'block';", input);

// Configure the client to upload local files to the remote Selenium instance
driver.setFileDetector(new LocalFileDetector());

// Local path on the machine running the tests — not inside the container!
input.sendKeys("/path/to/file/on/machine/which/runs/tests");
```

**Python**

```python
from selenium.webdriver.remote.file_detector import LocalFileDetector

input = driver.find_element(By.CSS_SELECTOR, "input[type='file']")
driver.execute_script("arguments[0].style.display = 'block';", input)
driver.file_detector = LocalFileDetector()
input.send_keys("/path/to/file/on/machine/which/runs/tests")
```

**WebdriverIO**

```javascript
const remoteFilePath = browser.uploadFile('/path/to/local/file');
$("input[type='file']").setValue(remoteFilePath);
```

## How it works

Not every driver implements the WebDriver upload endpoint — geckodriver,
IEDriver and `WebKitWebDriver` do not. WebSummoner fills that gap itself, so
upload behaves the same on every browser it ships.

**In Docker mode**, the hub unpacks the uploaded archive and copies the file
straight into the browser container, then hands the driver the resulting
in-container path. Nothing has to be shared between the hub and the container,
and the driver never needs its own upload support.

Files land in `/opt/websummoner/uploads` inside the container and disappear
with it. The file is written by a process running in the container, so it is
owned by the browser's own user and readable only by it (`0600`), and the
directory behaves like `/tmp` — sticky and world-writable. You may mount a
`tmpfs` over it if you want uploads to stay in memory.

The directory does not have to exist in the image — it is created on first
upload — so this works with custom and older browser images too. Images
without `tar` fall back to the Docker copy API automatically; the only thing
that mode cannot do is write into a mounted destination. Start the hub with:

```bash
./websummoner -enable-file-upload
```

**In [drivers mode](/guides/without-docker/)** there is no container, so the
hub writes the file to its own filesystem and returns that path:

```bash
./websummoner -disable-docker -enable-file-upload
```
