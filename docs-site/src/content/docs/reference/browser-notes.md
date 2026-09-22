---
title: Browser-specific behaviour
description: What differs between the browser images, and the configuration each one needs.
---

Every image passes the same
[container test suite](https://github.com/WebSummoner/websummoner-container-tests),
but the browsers are not identical. This page records each exception, what
WebSummoner already handles for you, and what your tests need to do.

| Browser | `browserName` | CDP and HAR | BiDi | Worth knowing |
| --- | --- | --- | --- | --- |
| Chrome | `chrome` | Yes | Yes | Accepts `CH_POLICY_` policies |
| Edge | `MicrosoftEdge` | Yes | Yes | Options go under `ms:edgeOptions` |
| Opera | `opera` | Yes | Yes | Hub adds to `goog:chromeOptions`; extra window handles |
| Brave | `brave` | Yes | Yes | Hub adds to `goog:chromeOptions`; Shields; accepts `CH_POLICY_` |
| Yandex | `yandex` | Yes | Yes | Hub adds to `goog:chromeOptions`; own start page |
| Firefox | `firefox` | No | Yes | Strict about certificates |
| WebKit | `safari` | No | No | Cookies, proxies and teardown differ |

Opera and Yandex gain CDP and HAR with their next image build; the images
published before it lack the helper that serves CDP.

## Chromium browsers

For Opera, Brave and Yandex the hub rewrites the session request: it sets
`browserName` to `chrome`, points the driver at the real browser binary and
adds the start-up arguments each browser needs. Your own `goog:chromeOptions`
are kept — arguments, `prefs` and extensions reach the browser — with your
arguments placed after the hub's, so yours win when both set the same switch.
A `binary` you set is replaced by the hub's.

Chrome and Brave accept Chromium enterprise policies per session through
`CH_POLICY_<Name>` variables; see
[Browser policies](/reference/capabilities/#browser-policies--ch_policy_name).

## Chrome

The reference image. Nothing beyond the capabilities documented elsewhere.

## Edge

Use `browserName: MicrosoftEdge` and Selenium's `EdgeOptions`. Edge reads its
arguments from `ms:edgeOptions`, so arguments set through `ChromeOptions` are
ignored.

## Brave

Brave runs as root inside its image. It downloads to
`/home/selenium/Downloads` like every other image, so
[file download](/guides/file-download/) works unchanged.

Brave Shields report a false screen size to web pages as a fingerprinting
defence: `screen.width` differs from the requested `screenResolution` even
though the display is correct, and it changes from site to site. Turn Shields
off for the session when a test needs the real value:

```json
{ "env": ["CH_POLICY_BraveShieldsDisabledForUrls=[\"*\"]"] }
```

Leave Shields on otherwise: it is how Brave behaves for its users.

## Yandex

Yandex opens its own start page (`https://ya.ru/`) shortly after launch, which
can replace the page the driver has just loaded. The hub starts Yandex with
`about:blank` as its home page; if the first navigation of a session is still
replaced, navigate to `about:blank` once before it.

Yandex does not start when a managed policy file is present, so the image has
no `CH_POLICY_` support.

## Opera

Opera ships Chromium **N+16** — Opera 134 is Chromium 150, Opera 135 is Chromium
151 — and `operachromiumdriver` release tags follow the **Chromium** line, not
Opera's. The build tool works this out and always pairs the browser with the
right driver.

Opera also publishes its driver late, so a new Opera can ship before the driver
for its Chromium line exists. The build tool handles that by falling back to the
**newest `operadriver`**, not to a Chrome-for-Testing chromedriver. The version
check in this driver family is a *warning*, not a refusal — a driver one line
behind drives the browser and logs `This version of OperaDriver has not been
tested with Opera version 151`, then works normally.

| Opera | Driver used | Container suite |
| --- | --- | --- |
| **135.0.5973.76** *(current)* | `OperaDriver 151` (matching line) | **full suite passes** |
| 134.0.5954.66 | `OperaDriver 150` (matching line) | **full suite passes** |

Substituting a chromedriver is the tempting shortcut here and it does start a
session, but it crashes the renderer whenever a *page* opens a window — a
`target="_blank"` link or `window.open()` ends the session with
`disconnected: Unable to receive message from renderer`. This is not a version
mismatch: Opera 135 ships Chromium 151.0.7922.176 and the substituted
chromedriver is that same build. Opera patches its Chromium, and only Opera's
own driver accounts for those patches. **A real driver one line behind beats a
foreign driver on the exact line.**

### Opera's own driver speaks JSONWP unless asked

`operadriver` answers a W3C session request in the legacy JSONWP dialect, which
a W3C-only client such as Selenium 4 cannot decode — `find_element` comes back
as `{'ELEMENT': ...}` rather than an element
([operachromiumdriver#96](https://github.com/operasoftware/operachromiumdriver/issues/96)).
It does support W3C, but only when asked. WebSummoner sets that for you, so
nothing is needed on the client side.

### Opera reports its interface as browser windows

A fresh Opera session already has several window handles, not one:

```text
chrome://startpageshared/      Speed Dial
chrome://address-bar-dropdown/ Address Bar Dropdown
chrome://startpage/            Speed Dial
```

They cannot be closed and no start-up flag removes them. Write window tests
against the *change* in handle count rather than an absolute number, and match
windows by title or URL rather than assuming the first handle is your page.

## Firefox

Firefox has no CDP endpoint, so HAR capture is not available; BiDi is. The
image relays geckodriver's BiDi port so the hub can reach it, which needs
nothing from the client.

Firefox applies the W3C default for `acceptInsecureCerts` strictly. Behind a
TLS-intercepting proxy, set it to `true`; Chromium drivers are more lenient.

## WebKit (Safari)

WebKit passes the full container suite, the same one every other image runs.
Getting there meant filling two `WebKitWebDriver` gaps from outside the browser:

- **File upload** works — the hub copies uploaded files straight into the
  container, so it does not depend on the driver implementing the endpoint.
- **The `proxy` capability** works — see [Proxies](#proxies-on-webkit) below.

Two behaviours still need care when you write tests: cookies must set
`sameSite`, and WebKit is the first engine to suffer when the host is loaded.

### Setting cookies

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

### Proxies on WebKit

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

### `quit()` can throw even though the session ended

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

### Other WebKit differences

- **Sound needs a click — automatic with `enableAudio`.** WebKit plays
  nothing without a user gesture; the image sends it after every
  navigation when `enableAudio` is set.
- **Use window size, not fullscreen.** The W3C fullscreen-window command
  does not complete on the WebKitGTK driver; `setWindowRect(0, 0, 1920, 1080)`
  achieves the same effect and works.
- **No `data:` URL navigation.** The driver rejects `data:` URLs — serve
  test pages over HTTP instead (a fixture server, or any web server).
- **`close()` ends the session**, reporting an empty-message error like its
  `quit()`. Use one window per session rather than closing them.

