#!/usr/bin/env bash
# Run from the repository root after checkout. No server or release is modified.
set -euo pipefail
version="$(tr -d '\r\n' < common/x_x_version.txt)"
official_pattern='v[0-9]+\.[0-9]+\.[0-9]+(-(alpha|beta|rc|patch)\.[0-9]+(-i18nfix\.[0-9]+)?)?'
custom_pattern="^${official_pattern}\\+lqy007700\\+v[1-9][0-9]*$"
legacy_pattern="^${official_pattern}-x-x\\.[1-9][0-9]*$"
if [[ "$version" =~ $custom_pattern ]]; then
  upstream="${version%+lqy007700+v*}"
  expected_tag="$version"
elif [[ "$version" =~ $legacy_pattern ]]; then
  upstream="${version%-x-x.*}"
  expected_tag="x-x-custom-${version}"
else
  printf 'Invalid X-X release version: %s\n' "$version" >&2
  exit 1
fi
commit="$(git rev-parse HEAD)"
if [[ -n "${RELEASE_TAG:-}" && "$RELEASE_TAG" != "$expected_tag" ]]; then
  printf 'Release tag must be %s\n' "$expected_tag" >&2
  exit 1
fi
printf 'version=%s\nupstream=%s\ncommit=%s\n' "$version" "$upstream" "$commit"
if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
  printf 'version=%s\nupstream=%s\ncommit=%s\n' "$version" "$upstream" "$commit" >> "$GITHUB_OUTPUT"
fi
