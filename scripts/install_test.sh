#!/usr/bin/env sh
# Unit tests for installer helpers. Does not download GitHub assets.
set -eu
cd "$(dirname "$0")/.."
KPOD_INSTALL_TEST=1
# shellcheck disable=SC1091
. ./scripts/install.sh

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

got=$(printf '%s' '{"tag_name":"v2.0.0","prerelease":false}' | kpod_tag_from_latest_json)
[ "$got" = "v2.0.0" ] || fail "json parse got ${got}"

got=$(printf '%s\n' '  "url": "https://api.github.com/repos/MohamedBasaleh/kpod/releases/1",' '  "tag_name": "v2.1.0",' | kpod_tag_from_latest_json)
[ "$got" = "v2.1.0" ] || fail "pretty json parse got ${got}"

kpod_validate_release_tag "v2.0.0" || fail "v2.0.0 should be valid"
kpod_validate_release_tag "v2.0.0-rc.2" || fail "v2.0.0-rc.2 should be valid"
kpod_validate_release_tag "releases" && fail "releases must be rejected"
kpod_validate_release_tag "latest" && fail "latest must be rejected"
kpod_validate_release_tag "" && fail "empty must be rejected"
kpod_validate_release_tag "v2" && fail "short tag must be rejected"

echo "install_test.sh ok"
