---
title: Browser images
description: Ready-made Docker images with browsers — where to find them and what is inside.
---

WebSummoner runs any image that speaks WebDriver, but you usually want the
maintained ones:

| Browser | Image | Notes |
| --- | --- | --- |
| Chrome | `websummoner/chrome` | Driver-based, `path: /` |
| Firefox | `websummoner/firefox` | Driver-based, `path: /` |
| Opera | `websummoner/opera` | Driver-based, `path: /` |
| Brave | `websummoner/brave` | Chromium-based, driver-based, `path: /` |
| Edge | `websummoner/edge` | Driver-based, `path: /` |
| Yandex Browser | `websummoner/yandex` | Driver-based, `path: /` |
| Safari (WebKit engine) | `websummoner/safari` | WebKitGTK build, `browserName: safari`, `path: /` |
| Video recorder | `websummoner/video-recorder` | Used automatically for `enableVideo` |

All images are free to use. Each browser is published under three tag
levels — a floating line tag, a major.minor alias and an immutable
full-version pin; see
[Image tags and versioning](/reference/image-tags/). Version coverage
starts from the releases that were current at WebSummoner's first
release (September 2026); legacy browser versions are not provided. For
example:

```bash
docker pull websummoner/chrome:152
docker pull websummoner/firefox:154
docker pull websummoner/edge:152
docker pull websummoner/opera:135
docker pull websummoner/yandex:26.6
docker pull websummoner/brave:1.94
```

## Browser-specific behaviour

All seven browsers pass the whole
[container test suite](https://github.com/WebSummoner/websummoner-container-tests).
Getting two of them there took work that shows through to you, so this section
records what that was and the few behaviours worth knowing before you write
tests against them.

### Opera

Opera ships Chromium **N+16** — Opera 134 is Chromium 150, Opera 135 is Chromium
151 — and `operachromiumdriver` release tags follow the **Chromium** line, not
Opera's. The build tool works this out and always pairs the browser with the
right driver.

Opera also publishes its driver late: at the time of writing there is no
driver on the Chromium 151 line that Opera 135 is built from. The build tool
handles this by falling back to the **newest `operadriver`**, not to a
Chrome-for-Testing chromedriver. The version check in this driver family is a
*warning*, not a refusal — `OperaDriver 150` drives Opera 135 and logs
`This version of OperaDriver has not been tested with Opera version 151`, then
works normally.

| Opera | Driver used | Container suite |
| --- | --- | --- |
| **135.0.5973.66** *(current)* | `OperaDriver 150` (newest published) | **all 32 tests pass** |
| 134.0.5954.66 | `OperaDriver 150` (matching line) | **all 32 tests pass** |

Substituting a chromedriver is the tempting shortcut here and it does start a
session, but it crashes the renderer whenever a *page* opens a window — a
`target="_blank"` link or `window.open()` ends the session with
`disconnected: Unable to receive message from renderer`. This is not a version
mismatch: Opera 135 ships Chromium 151.0.7922.176 and the substituted
chromedriver is that same build. Opera patches its Chromium, and only Opera's
own driver accounts for those patches. **A real driver one line behind beats a
foreign driver on the exact line.**

#### Opera's own driver speaks JSONWP unless asked

`operadriver` answers a W3C session request in the legacy JSONWP dialect, which
a W3C-only client such as Selenium 4 cannot decode — `find_element` comes back
as `{'ELEMENT': ...}` rather than an element
([operachromiumdriver#96](https://github.com/operasoftware/operachromiumdriver/issues/96)).
It does support W3C, but only when asked. WebSummoner sets that for you, so
nothing is needed on the client side.

#### Opera reports its interface as browser windows

A fresh Opera session already has several window handles, not one:

```text
chrome://startpageshared/      Speed Dial
chrome://address-bar-dropdown/ Address Bar Dropdown
chrome://startpage/            Speed Dial
```

They cannot be closed and no start-up flag removes them. Write window tests
against the *change* in handle count rather than an absolute number, and match
windows by title or URL rather than assuming the first handle is your page.

### WebKit (Safari)

`WebKitWebDriver` is less complete than the Chromium and Gecko drivers, and
WebSummoner fills two of the gaps from outside the browser:

- **File upload** works — the hub copies uploaded files straight into the
  container, so it does not depend on the driver implementing the endpoint.
- **The `proxy` capability** works — see [Proxies](#proxies-on-webkit) below.

Two things still need care: cookies must set `sameSite`, and WebKit is sensitive
to host load.

#### Setting cookies

`addCookie` works, but the cookie **must set `sameSite`**. Chromium and Gecko
apply a default when the attribute is missing; `WebKitWebDriver` drops the
cookie instead — and answers the command with success, so nothing surfaces until
a later read comes back empty.

```java
Cookie cookie = new Cookie.Builder("name", "value")
        .path("/")
        .sameSite("Lax")      // required on WebKit, harmless elsewhere
        .build();
driver.manage().addCookie(cookie);
```

`Lax` and `Strict` both work. `None` is rejected over plain HTTP on every engine,
since it requires `Secure`.

Setting `sameSite` explicitly is good practice anyway, so the portable form costs
nothing.

#### Proxies on WebKit

`WebKitWebDriver` does not implement the `proxy` capability — SafariDriver
rejects it outright (*"Capability 'proxy' could not be honored"*) and WebKitGTK
silently ignores it. WebKit takes its proxy from the system, which on Linux means
GLib's proxy resolver, and inside a container there is no desktop configuration
for it to read, so every request resolves to *direct*.

WebSummoner translates the capability for you: when a WebKit session asks for a
manual proxy, the hub sets the standard `http_proxy`, `https_proxy` and
`no_proxy` variables on the browser container, which is where WebKit picks its
proxy up. Use the standard capability and it works:

```java
Proxy proxy = new Proxy()
        .setProxyType(Proxy.ProxyType.MANUAL)
        .setHttpProxy("proxy.example.com:8080")
        .setSslProxy("proxy.example.com:8080");
options.setCapability("proxy", proxy);
```

`localhost` and `127.0.0.1` are always excluded — the driver reaches the browser
over that connection, and proxying it stops the session starting at all.

#### `quit()` can throw even though the session ended

`WebKitWebDriver` ends the session and then closes the connection **without
sending a response**, so the client raises an empty-message `WebDriverException`
for a teardown that actually succeeded. Nothing is left behind — WebSummoner
logs `SESSION_DELETED` and `CONTAINER_REMOVED` either way, and no container
survives it.

Selenium's own WebKitGTK binding used to swallow this exception for the same
reason. Do the same in your teardown, and keep it scoped to WebKit so that a
failing `quit()` stays a real failure everywhere else:

```java
try {
    driver.quit();
} catch (WebDriverException e) {
    if (!"safari".equals(browserName)) {
        throw e;
    }
    // WebKit closes the socket after ending the session; nothing leaked.
}
```

WebKitGTK is also sensitive to host load. Give it an otherwise quiet grid when
a run matters, and clean up leaked containers — sessions abandoned without
`quit()` hold slots against `-limit` and the contention surfaces on WebKit
first.

## Where things live

- Build files and per-browser documentation (exact versions, bundled
  drivers): the [images repository](https://github.com/WebSummoner/images).
- New browser versions are provisioned and published to Docker Hub by
  RIADVICE as they stabilize. Custom builds are possible with the public
  [`images` tool](https://github.com/WebSummoner/images#building-images).

## VNC support

VNC is built into every image: an `x11vnc` server streams the browser
screen whenever the session is started with the `enableVNC` capability
(or `ENABLE_VNC=true` when running the image standalone). It is required
for the live view in
[WebSummoner UI](https://github.com/WebSummoner/websummoner-ui). There are
no separate VNC images.

## Safari note

Real Safari only runs on macOS and iOS. `websummoner/safari` is built from
the official WebKitGTK releases — the same engine Safari uses — compiled
with its matching WebKitWebDriver, so behavior is functionally equivalent
to Safari, though fonts and pixel-perfect rendering can differ from the
macOS browser. The `browserName` capability is `safari`; versions follow
the WebKitGTK release numbering rather than Safari marketing numbers.
Safari is tagged with the full WebKitGTK version only (`safari:2.52.6`) —
there is no floating line tag, because each WebKitGTK release is a
substantially different engine:

```json
{
  "safari": {
    "default": "2.52.6",
    "versions": {
      "2.52.6": {
        "image": "websummoner/safari:2.52.6",
        "port": "4444",
        "path": "/"
      }
    }
  }
}
```

Testing the actual macOS Safari requires Apple hardware — WebSummoner can
drive it through the [standalone driver
binaries](/reference/browsers-config/#standalone-driver-binaries) form of
`browsers.json` on a Mac running `safaridriver`.

### Safari-specific behavior

- **Sound needs a click — automatic with `enableAudio`.** WebKit plays
  nothing without a user gesture; the image sends it after every
  navigation when `enableAudio` is set.
- **Use window size, not fullscreen.** The W3C fullscreen-window command
  does not complete on the WebKitGTK driver; `setWindowRect(0, 0, 1920, 1080)`
  achieves the same effect and works.
- **No `data:` URL navigation.** The driver rejects `data:` URLs — serve
  test pages over HTTP instead (a fixture server, or any web server).

## Custom root certificates

On corporate networks the tested environment often uses TLS certificates
from a private root CA. The standard `acceptInsecureCerts` capability
ignores certificate errors, but does not help with HSTS. Instead, add your
root certificate to an image at container start with environment variables —
one variable per certificate, holding the Base64-encoded `cert.pem`:

```bash
CERT_CONTENTS=$(cat cert.pem | base64 -w0)   # macOS: base64
docker run -e ROOT_CA_MY_CERT="$CERT_CONTENTS" ... websummoner/chrome:152
```

The variable suffix (`MY_CERT` above) becomes the certificate name in the
browser certificate storage.

## Custom browser profile in Chrome

When launching Chrome with a custom profile directory, DevTools do not work
unless `BROWSER_PROFILE_DIR` is also set to the same directory:

```json
{
  "capabilities": {
    "alwaysMatch": {
      "browserName": "chrome",
      "goog:chromeOptions": {
        "args": ["user-data-dir=/profiles/custom.XYZ"]
      },
      "websummoner:options": {
        "env": ["BROWSER_PROFILE_DIR=/profiles/custom.XYZ"]
      }
    }
  }
}
```

## Audio

Every standard image runs a PulseAudio server in the container. Video
recordings therefore include an audio track automatically — see
[Recording audio](/guides/video-recording/#recording-audio).

## Custom images

Any Docker image works as long as it starts a WebDriver-compatible service on
a known port — see the `image`, `port` and `path` fields in
[Browsers configuration](/reference/browsers-config/). To build your own
browser images with the public `images` tool, see
[Building browser images](/reference/building-browser-images/).
