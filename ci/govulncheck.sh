#!/bin/bash
#
# govulncheck with a narrow, documented allowlist.
#
# GO-2026-4887 and GO-2026-4883 are Moby daemon-side flaws (AuthZ plugin bypass
# and a plugin privilege off-by-one). WebSummoner embeds the Docker *client*
# and never runs that code, but both advisories are "unreviewed" and list no
# affected symbols, so govulncheck cannot narrow them and flags every package
# init instead. github.com/docker/docker has no fixed version and never will —
# the fix lives in github.com/moby/moby/v2, still in beta. Revisit when that
# module reaches a stable release.
#
# Anything not listed here still fails the build.
set -e

ALLOW="GO-2026-4887 GO-2026-4883"

go install golang.org/x/vuln/cmd/govulncheck@latest
GOVULNCHECK="$(go env GOPATH)/bin/govulncheck"

# Human-readable output for the log, then the machine-readable pass for the gate.
"$GOVULNCHECK" -tags production ./... || true

"$GOVULNCHECK" -format json -tags production ./... > /tmp/govuln.json 2>/dev/null || true

ALLOW="$ALLOW" python3 - /tmp/govuln.json <<'PY'
import json, os, sys

allow = set(os.environ["ALLOW"].split())
raw = open(sys.argv[1], encoding="utf-8").read()

# govulncheck emits pretty-printed objects back to back, not JSON lines.
dec, idx, called = json.JSONDecoder(), 0, {}
while idx < len(raw):
    while idx < len(raw) and raw[idx].isspace():
        idx += 1
    if idx >= len(raw):
        break
    obj, idx = dec.raw_decode(raw, idx)
    f = obj.get("finding")
    if not f:
        continue
    trace = f.get("trace") or []
    # A finding with a function in its trace is reachable from our code;
    # without one it is only an imported or required module.
    if trace and trace[0].get("function"):
        called.setdefault(f["osv"], set()).add(trace[0].get("module", "?"))

unexpected = {k: v for k, v in called.items() if k not in allow}
for osv in sorted(called):
    state = "ALLOWED" if osv in allow else "NEW"
    print(f"  {state:8} {osv}  ({', '.join(sorted(called[osv]))})")

if unexpected:
    print(f"\ngovulncheck: {len(unexpected)} vulnerability(ies) in called code", file=sys.stderr)
    sys.exit(1)
print("\ngovulncheck: no unexpected vulnerabilities in called code")
PY
