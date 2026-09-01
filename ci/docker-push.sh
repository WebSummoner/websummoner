#!/bin/bash

set -e

# Docker rejects upper-case repository names; the org has capitals.
IMAGE=$(echo "$GITHUB_REPOSITORY" | tr '[:upper:]' '[:lower:]')

[ -n "${DOCKER_USERNAME:-}" ] && [ -n "${DOCKER_PASSWORD:-}" ] || { echo "DOCKER_USERNAME and DOCKER_PASSWORD must be set" >&2; exit 1; }
printf '%s' "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin
docker buildx build --pull --push -t ""$IMAGE"" -t ""$IMAGE:$1"" --platform linux/amd64,linux/arm64 .
