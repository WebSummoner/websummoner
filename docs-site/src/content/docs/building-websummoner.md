---
title: Building WebSummoner
description: Build the binary and images using only Docker — no Go toolchain required.
---

Following the same philosophy as
[golang-docker-boilerplate](https://github.com/riadvice/golang-docker-boilerplate),
every build task runs inside a pinned Go container (`golang:1.27-trixie` by
default). **No Go installation is needed on your machine — only Docker.**

## Everyday commands

Run each task in the pinned Go container. This repository is a self-contained
Go module, so mounting it alone is enough.

Compile for your platform:

```bash
docker run --rm -v "$PWD":/app -w /app golang:1.27-trixie \
    go build -buildvcs=false -o dist/websummoner .
```

Run the tests, with the same build tags CI uses:

```bash
docker run --rm -v "$PWD":/app -w /app golang:1.27-trixie \
    go test -tags 's3 metadata' -race ./...
```

Vet and check formatting:

```bash
docker run --rm -v "$PWD":/app -w /app golang:1.27-trixie \
    bash -c 'go vet ./... && test -z "$(gofmt -l .)"'
```

Lint with the project's configuration:

```bash
docker run --rm -v "$PWD":/app -w /app golangci/golangci-lint:latest \
    golangci-lint run ./...
```

## Release binaries and the Docker image

`ci/build.sh` cross-compiles every release target into `dist/`, and CI runs it
for tagged releases — so released binaries always come from one place. To
reproduce a release build locally, run that script in the container:

```bash
docker run --rm -v "$PWD":/app -w /app golang:1.27-trixie ci/build.sh
```

Then build the image from the linux binary for your architecture:

```bash
docker build --build-arg TARGETARCH=amd64 -t websummoner:local .
```

## Browser images

Ready-made browser images are published to
[Docker Hub](https://hub.docker.com/u/websummoner) by RIADVICE and updated as
new browser versions stabilize — you never need to build them to use
WebSummoner. Just pull:

```bash
docker pull websummoner/chrome:152.0
docker pull websummoner/firefox:154.0
```

To build fully custom images yourself, the
[images repository](https://github.com/WebSummoner/images) contains the
public `images` build tool and its
[build documentation](https://github.com/WebSummoner/images#building-images).

## Documentation site

This documentation is an [Astro Starlight](https://starlight.astro.build)
site in `docs-site/`:

```bash
cd docs-site
npm install
npm run dev       # live preview at http://localhost:4321/
npm run build     # static output in dist/ + llms.txt
```

It is built and deployed to GitHub Pages automatically by the custom
`docs` GitHub Actions workflow.
