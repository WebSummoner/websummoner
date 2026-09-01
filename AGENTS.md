# AGENTS.md

## What this project is

WebSummoner is a Selenium hub: it receives WebDriver requests and starts a
fresh browser in a Docker container for each session, then proxies the session
to it. One container per session means no state leaks between tests, and the
container is destroyed when the session ends.

It is a maintained fork of [Aerokube Selenoid](https://github.com/aerokube/selenoid),
which its maintainers archived in December 2024.

**The goal is to be boring and fast.** A ~10 MB Go binary, no JVM, no database,
starting browsers in about a second. Selenium Grid is the feature-rich
alternative; WebSummoner competes on weight and operational simplicity. Changes
that add moving parts to the hot path need a strong reason.

Two commitments shape most decisions:

- **Compatibility with Selenoid is deliberate.** `selenoid:options`, the
  `X-Selenoid-*` headers and the `/aerokube/...` vendor path all still work.
  Migration should be a one-line change for users. Do not remove these.
- **Browser quirks are absorbed by the hub, not pushed onto users.** Where a
  driver is incomplete, the fix belongs in the hub so it applies to images
  built long before it.

## Workspace layout

WebSummoner is eight sibling repositories, not a monorepo. Each one is a
self-contained Go module that builds on its own — cross-repo dependencies are
resolved from published pseudo-versions, not `replace` directives.

```
websummoner/                 the hub (this repository)
ggr/                         load balancer — the hub and ggr-ui depend on it
ggr-ui/                      ggr's UI
websummoner-ui/              the session UI (React app embedded into a Go binary)
images/                      browser image Dockerfiles + the Go build tool
cm/                          the installer/manager CLI
websummoner-container-tests/ the JUnit 5 cross-browser suite
websummoner-website/         the marketing site
```

Mounting a single repository is enough:

```bash
docker run --rm -v "$PWD":/app -w /app golang:1.27 go build -buildvcs=false ./...
```

Mounting the workspace root also works, but the working directory must then be
the subproject — `-v "$PWD":/ws -w /ws/websummoner` from the **workspace root**,
not from inside `websummoner/`, which gives "no Go files".

:warning: Do not reintroduce a `replace` directive pointing at a sibling. It
works locally and breaks CI, which clones one repository: `replacement
directory ../ggr does not exist`. This bit three repositories at once.

## The toolchain is dockerised on purpose

There is no host Go, JDK, Node, or SDKMAN, and none should be installed.

| Language | Image |
| --- | --- |
| Go | `golang:1.27` (`golang:1.27-trixie` for image builds) |
| Java | `maven:3-eclipse-temurin-25` with the `ws-m2` volume at `/root/.m2` |
| Node | `node:24` |

Build the hub binary with **`CGO_ENABLED=0`**. The hub image is `FROM scratch`;
a dynamically linked binary fails at runtime with the misleading
`exec /usr/bin/websummoner: no such file or directory`.

CI builds releases with `ci/build.sh` and runs tests with `ci/test.sh` — those
are the source of truth. There is no Makefile.

## Running the browser tests

```bash
cd websummoner-images-build
./run-container-tests.sh                 # every browser the grid advertises
./run-container-tests.sh safari          # one browser — positional, not TESTS=
TESTS=TestProxy ./run-container-tests.sh safari
```

`TESTS=` filters **test classes**, not browsers. The browser is positional.

**Evaluate results from isolated runs.** Under a full concurrent sweep the host
becomes the variable: WebKit degrades first, and Chromium browsers occasionally
fail an alert test that passes every isolated re-run.

## Where behaviour lives

`adaptDriverCapabilities` in `websummoner.go` is the single place browser
capabilities are translated, and it runs in the hub — so a browser fix applies
to images built long before it, with no rebuild. Prefer fixing a browser there
over changing an image.

`processBody` wraps legacy JSONWP driver replies in the W3C envelope modern
clients expect. If you change the session-response shape, the hub tests that
decode it must change with it.

## Things that will waste your time

**Leaked sessions starve the grid.** A session created by hand and not deleted
leaves a container running *and* a phantom slot in the hub's in-memory count —
`/status` shows `used: 3` with no containers. At `-limit 5` a few of these
starve the grid, and WebKit fails first. `docker restart ws-hub` clears the
count. Always `DELETE` sessions you create.

**Do not `pkill -f` on a pattern matching your own command line.** It kills the
shell running it. Find the PID another way.

**The UI is a Go binary with the React app embedded**, not a static site. The
image is `FROM scratch` and copies a pre-built binary, so `docker build` alone
packages whatever binary is on disk. The full chain is:

```bash
# 1. React build   2. statik embed   3. go build   4. docker build
docker run --rm -v "$PWD/websummoner-ui/ui":/app -w /app node:24 \
  sh -c 'CI=true npx react-scripts build'
docker run --rm -v "$PWD":/ws -w /ws/websummoner-ui golang:1.27 sh -c '
  go install github.com/rakyll/statik@latest
  PATH=$PATH:/root/go/bin go generate ./...
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -o websummoner-ui .'
docker build -t websummoner/websummoner-ui:latest websummoner-ui
```

**Truncating a log file another process holds open does not reset it.** The
writer keeps its offset and you get a sparse file, so counts read as zero. Use a
fresh filename per measurement. Two conclusions in this repository's history
were wrong because of exactly this.

## Style

Match the surrounding code: its naming, its idiom, and its comment density.
Comments explain **why**, not what, and stay short — one or two lines. A
maintainer already knows what the API does; write only the part that is not
obvious from the code. Do not restate the line below.

Go code is `gofmt -s` clean and passes `golangci-lint` with the repository's
`.golangci.yml`. Java goes through Error Prone and Spotless, so formatting is
enforced — run `mvn -B -ntp verify -Dtest='*Test'` before claiming a Java
change compiles. An edit that looks like a no-op may be one: Spotless may have
rewrapped the line you were matching against.

## Documentation

Docs are an Astro Starlight site in `docs-site/`, not the deleted `docs/*.adoc`
files. Build with `npm run build` in `node:24`; it should report 30 pages.

Hand-written root-relative links are rewritten with the site's `base` by
`plugins/base-links.mjs`, so write `/reference/cli-flags/`, never
`/websummoner/reference/cli-flags/`.

When a fix changes browser behaviour, update `reference/browser-images.md` in
the same change. Users hit these behaviours before they read the code.

## Rules

**Never commit or push unless explicitly asked.** This workspace carries a
large amount of deliberate uncommitted work across eight repositories, and a
helpful `git commit` destroys the reviewer's ability to see what changed.

**Never add build output to git.** `dist/`, compiled helper binaries and
`node_modules/` are ignored in every repository. When you add a new helper
binary, add it to `.gitignore` in the same change — three have been missed this
way already.

**Verify, do not assume.** Run the command, read the output, check the version
against the registry. Several long detours in this project came from a
plausible explanation that turned out to be wrong.
