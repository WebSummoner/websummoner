# Security policy

## Supported versions

Security fixes land on the latest release. Older tags are not patched.

## Reporting a vulnerability

Open a private security advisory on GitHub (**Security → Report a
vulnerability**), or email <websummoner@riadvice.com>.

Please include reproduction steps and the affected version. We aim to
acknowledge within three working days and to publish a fix or a mitigation
before the advisory goes public.

Do not open a public issue for a suspected vulnerability.

## Scope

WebSummoner starts browsers in Docker containers on behalf of WebDriver
clients. It talks to the Docker daemon directly, so **anyone who can reach the
hub can start containers**. Treat the listening port as privileged: put it
behind authentication, a private network, or [GGR](https://github.com/WebSummoner/ggr).

## Dependency scanning

Every CI run executes `govulncheck`, and Dependabot tracks module updates.
Release binaries are built in a pinned `golang:1.27-trixie` container so they
always come from a known compiler.
