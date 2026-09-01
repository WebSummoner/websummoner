---
title: Contributing
description: How to build, test and send changes for WebSummoner.
---

Contributions are welcome — issues, documentation improvements and code.

## Development quick start

You need only Docker — see [Building WebSummoner](/building-websummoner/)
for the full picture:

```bash
git clone https://github.com/WebSummoner/websummoner.git
cd websummoner

# tests, with the build tags CI uses
docker run --rm -v "$PWD/..":/ws -w /ws/websummoner golang:1.27-trixie \
    go test -tags 's3 metadata' -race ./...

# vet and formatting
docker run --rm -v "$PWD/..":/ws -w /ws/websummoner golang:1.27-trixie \
    bash -c 'go vet ./... && test -z "$(gofmt -l .)"'
```

See [Building WebSummoner](/building-websummoner/) for the full set.

## Sending changes

1. Fork the repository and create a topic branch.
2. Make your change; add or adapt tests where reasonable.
3. Ensure the vet, formatting and test commands above pass — CI runs the same
   checks on every pull request.
4. Open a pull request describing **what** changes and **why**.

For docs changes, edit the Markdown pages under `docs-site/src/content/docs/`
and check them with `npm run dev` inside `docs-site/` — a live preview shows
exactly how the page will render.

## Code of conduct

Be respectful. See `CODE_OF_CONDUCT.md` in the repository root.

## Credits

WebSummoner is developed and maintained by
[RIADVICE](https://riadvice.com).
