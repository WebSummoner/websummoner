#!/bin/bash

set -e

# Docker rejects upper-case repository names; the org has capitals.
IMAGE=$(echo "$GITHUB_REPOSITORY" | tr '[:upper:]' '[:lower:]')

# One build for every tag: buildx stamps provenance with the build time, so a
# second build of the same source lands on a different digest.
TAGS=(-t "$IMAGE")
for tag in "$@"; do TAGS+=(-t "$IMAGE:$tag"); done

[ -n "${DOCKER_USERNAME:-}" ] && [ -n "${DOCKER_PASSWORD:-}" ] || { echo "DOCKER_USERNAME and DOCKER_PASSWORD must be set" >&2; exit 1; }
printf '%s' "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin
docker buildx build --pull --push "${TAGS[@]}" --platform linux/amd64,linux/arm64 .
