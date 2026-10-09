#!/usr/bin/env bash
# Runs the ADL LRS conformance test suite against a local LRSolid.
# Requires Node.js (and git) or a local Docker image named
# adlnet/lrs-conformance-test-suite.
#
#   AUTH=keyId:secret VERSION=1.0.3 make xapi conformance
#   AUTH=keyId:secret VERSION=2.0.0 ENDPOINT=http://127.0.0.1:8080/xapi make xapi conformance
set -euo pipefail

VERSION="${VERSION:-1.0.3}"
ENDPOINT="${ENDPOINT:-http://127.0.0.1:8080/xapi}"
AUTH="${AUTH:-}"

if [[ -z "$AUTH" || "$AUTH" != *:* ]]; then
  echo "Set AUTH=keyId:secret (an xAPI client with scope \"all\")" >&2
  exit 1
fi

USER_NAME="${AUTH%%:*}"
PASSWORD="${AUTH#*:}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/tmp/conformance"
mkdir -p "$OUT"

if command -v docker >/dev/null 2>&1 && docker image inspect adlnet/lrs-conformance-test-suite >/dev/null 2>&1; then
  docker run --rm --network host \
    -e LRS_ENDPOINT="$ENDPOINT" \
    -e LRS_USERNAME="$USER_NAME" \
    -e LRS_PASSWORD="$PASSWORD" \
    -e XAPI_VERSION="$VERSION" \
    -v "$OUT:/out" \
    adlnet/lrs-conformance-test-suite
  exit $?
fi

DIR="$ROOT/tmp/lrs-conformance-test-suite"
if [[ ! -d "$DIR/.git" ]]; then
  git clone --depth 1 https://github.com/adlnet/lrs-conformance-test-suite.git "$DIR"
fi

cd "$DIR"
# Prefer a branch or tag that matches the requested xAPI version when one exists.
if git ls-remote --exit-code --heads origin "$VERSION" >/dev/null 2>&1; then
  git fetch --depth 1 origin "$VERSION"
  git checkout -B "$VERSION" "origin/$VERSION"
elif git ls-remote --exit-code --tags origin "$VERSION" >/dev/null 2>&1; then
  git fetch --depth 1 origin "refs/tags/$VERSION"
  git checkout "$VERSION"
fi

if [[ -f package-lock.json ]]; then
  npm ci
else
  npm install
fi

set +e
node bin/console.js lrs --endpoint "$ENDPOINT" --basicAuth "$USER_NAME:$PASSWORD" --version "$VERSION"
status=$?
if [[ $status -ne 0 ]]; then
  node bin/console.js lrs -e "$ENDPOINT" -u "$USER_NAME" -p "$PASSWORD" -a
  status=$?
fi
set -e

echo "Conformance finished with status $status. Logs, if the suite wrote any, are under $OUT."
exit "$status"
