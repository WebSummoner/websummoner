# WebSummoner
[![Build](https://github.com/WebSummoner/websummoner/actions/workflows/build.yml/badge.svg)](https://github.com/WebSummoner/websummoner/actions/workflows/build.yml)
[![Lint](https://github.com/WebSummoner/websummoner/actions/workflows/lint.yml/badge.svg)](https://github.com/WebSummoner/websummoner/actions/workflows/lint.yml)
[![codecov](https://codecov.io/gh/websummoner/websummoner/graph/badge.svg)](https://codecov.io/gh/websummoner/websummoner)
[![Release](https://img.shields.io/github/v/release/WebSummoner/websummoner)](https://github.com/WebSummoner/websummoner/releases/latest)
[![Docker Pulls](https://img.shields.io/docker/pulls/websummoner/websummoner)](https://hub.docker.com/r/websummoner/websummoner)

**WebSummoner is developed and maintained by [RIADVICE](https://riadvice.com)** — a Selenium hub updated with current Go toolchains, modern browser images and community fixes.

WebSummoner is a powerful implementation of [Selenium](http://github.com/SeleniumHQ/selenium) hub using [Docker](https://docker.com/) containers to launch browsers.
![WebSummoner Animation](docs-site/public/img/websummoner-animation.gif)

## Features

### One-command Installation
Start browser automation in minutes by downloading [Configuration Manager](https://github.com/WebSummoner/cm/releases) binary and running just **one command**:
```
$ ./cm websummoner start --vnc --tmpfs 128
```
The legacy `cm selenoid` subcommand is still accepted.
**That's it!** You can now use WebSummoner instead of Selenium server. Specify the following Selenium URL in tests:
```
http://localhost:4444/wd/hub
```

### Ready to use Browser Images
No need to manually install browsers or dive into WebDriver documentation. Available images:
![Browsers List](docs-site/public/img/browsers-list.gif)

New images are added right after official releases. You can create your custom images with browsers.

### Live Browser Screen and Logs
New **[rich user interface](https://github.com/WebSummoner/websummoner-ui)** showing browser screen and Selenium session logs:
![WebSummoner UI](docs-site/public/img/websummoner-ui.png)

### Video Recording
* Any browser session can be saved to [H.264](https://en.wikipedia.org/wiki/H.264/MPEG-4_AVC) video ([example](https://www.youtube.com/watch?v=maB298oO5cI))
* An API to list, download and delete recorded video files

### Convenient Logging

* Any browser session logs are automatically saved to files - one per session
* An API to list, download and delete saved log files

### Lightweight and Lightning Fast
Suitable for personal usage and in big clusters:
* Consumes **10 times** less memory than Java-based Selenium server under the same load
* **Small 6 Mb binary** with no external dependencies (no need to install Java)
* **Browser consumption API** working out of the box
* Ability to send browser logs to **centralized log storage** (e.g. to the [ELK-stack](https://logz.io/learn/complete-guide-elk-stack/))
* Fully **isolated** and **reproducible** environment

### Detailed Documentation
* Detailed [documentation](https://websummoner.github.io/websummoner/) — built from [`docs-site/`](docs-site/) with [Astro Starlight](https://starlight.astro.build), deployed to GitHub Pages by the custom `docs` workflow
* Compatible with the vast [Selenoid](https://stackoverflow.com/questions/tagged/selenoid) community knowledge on StackOverflow

## Complete Guide & Build Instructions

Complete reference guide (including building instructions) lives at [websummoner.github.io/websummoner](https://websummoner.github.io/websummoner/) (source: [`docs-site/`](docs-site/)).

Building requires **only Docker** — no Go installation on your machine:
```
$ docker run --rm -v "$PWD":/app -w /app golang:1.27-trixie \
      go test -tags 's3 metadata' -race ./...
```

## Migrating from Selenoid

WebSummoner keeps backwards compatibility with Selenoid client-facing APIs:
* The legacy `selenoid:options` capability prefix is still accepted (the new `websummoner:options` takes precedence when both are set)
* The legacy `X-Selenoid-No-Wait` and `X-Selenoid-File` headers and the `/aerokube/...` proxy path fragment still work
* Inside the Docker image the configuration file moved from `/etc/selenoid/browsers.json` to `/etc/websummoner/browsers.json` (override with the `-conf` flag or your own entrypoint)

## WebSummoner in Kubernetes

WebSummoner was initially created to be deployed on hardware servers or virtual machines and is not suitable for Kubernetes.

## Known Users

WebSummoner runs browser automation for engineering teams all over the world — from startups to Fortune 500 companies — and workflows built for its compatible predecessors carry over unchanged:

[![JetBrains](docs-site/public/img/logo/jetbrains.png)](http://jetbrains.com/) [![Yandex](docs-site/public/img/logo/yandex.png)](https://yandex.com/company/) [![Sberbank Technology](docs-site/public/img/logo/sbertech.png)](http://sber-tech.com/) [![ThoughtWorks](docs-site/public/img/logo/thoughtworks.png)](https://thoughtworks.com/) [![VK.com](docs-site/public/img/logo/vk.png)](https://vk.com/) [![SuperJob](docs-site/public/img/logo/superjob.png)](http://superjob.ru/) [![PropellerAds](docs-site/public/img/logo/propellerads.png)](http://propellerads.com/) [![AlfaBank](docs-site/public/img/logo/alfabank.png)](https://alfabank.com/) [![3CX](docs-site/public/img/logo/3cx.png)](https://www.3cx.com/) [![IQ Option](docs-site/public/img/logo/iq_option.png)](https://iqoption.com/) [![Mail.Ru Group](docs-site/public/img/logo/mail_ru.png)](https://corp.mail.ru/en/) [![Newegg.Com](docs-site/public/img/logo/newegg.png)](http://newegg.com/) [![Badoo](docs-site/public/img/logo/badoo.png)](https://badoo.com/team/) [![BCS](docs-site/public/img/logo/bcs.png)](https://bcs.ru/) [![Quality Lab](docs-site/public/img/logo/quality-lab.png)](https://quality-lab.ru) [![AT Consulting](docs-site/public/img/logo/at-consulting.png)](https://www.at-consulting.ru/) [![Royal Caribbean International](docs-site/public/img/logo/royal-caribbean.png)](https://www.royalcaribbean.com/) [![Sixt](docs-site/public/img/logo/sixt.png)](https://sixt.com/) [![Testjar](docs-site/public/img/logo/testjar.png)](http://www.testjar.com/) [![Flipdish](docs-site/public/img/logo/flipdish.png)](https://www.flipdish.com/) [![RIADVICE](docs-site/public/img/logo/riadvice.svg)](https://riadvice.com/)

## License

Apache 2.0 — see [LICENSE](LICENSE).
