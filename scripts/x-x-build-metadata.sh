#!/usr/bin/env bash
# Run from the repository root after checkout. No server or release is modified.
set -euo pipefail
version="$(tr -d '\r\n' < common/x_x_version.txt)"
if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-(alpha|beta|rc|patch)\.[0-9]+(-i18nfix\.[0-9]+)?)?-x-x\.[1-9][0-9]*$ ]]; then
  printf 'Invalid X-X release version: %s\n' "$version" >&2
  exit 1
fi
commit="$(git rev-parse HEAD)"
upstream="${version%-x-x.*}"
if [[ -n "${RELEASE_TAG:-}" && "$RELEASE_TAG" != "x-x-custom-${version}" ]]; then
  printf 'Release tag must be x-x-custom-%s\n' "$version" >&2
  exit 1
fi
printf 'version=%s\nupstream=%s\ncommit=%s\n' "$version" "$upstream" "$commit"
if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
  printf 'version=%s\nupstream=%s\ncommit=%s\n' "$version" "$upstream" "$commit" >> "$GITHUB_OUTPUT"
fi
