#!/usr/bin/env bash
# Usage: wait-for-npm.sh <package> <version>
#
# Waits until npm serves https://registry.npmjs.org/<package>/<version>, the
# version document the MCP Registry fetches to validate server.json. npm
# accepts a publish with a 202 and can take minutes to serve the new version
# (two to ten minutes in practice), and the registry refuses server.json with a 404
# until it does.
#
# WAIT_TIMEOUT (seconds, default 900) caps the wait; WAIT_INTERVAL (seconds,
# default 15) sets the pause between checks.
set -euo pipefail

pkg="${1:-}"
version="${2:-}"
# jq -r prints "null" for a missing key; polling for null@null would only
# fail 15 minutes later.
for arg in "$pkg" "$version"; do
  if [ -z "$arg" ] || [ "$arg" = null ]; then
    echo "::error::usage: wait-for-npm.sh <package> <version> (got '$pkg' '$version'); check server.json"
    exit 2
  fi
done
timeout="${WAIT_TIMEOUT:-900}"
interval="${WAIT_INTERVAL:-15}"
waited=0

while :; do
  # The same URL and Accept header as the registry's validator. The package
  # document that `npm view` reads is cached apart from it, so its answer
  # proves nothing about this one.
  got=$(curl -fsS --max-time 30 -H 'Accept: application/json' "https://registry.npmjs.org/$pkg/$version" 2>/dev/null \
    | jq -r .version 2>/dev/null || true)
  if [ "$got" = "$version" ]; then
    echo "npm serves $pkg@$version (waited ${waited}s)"
    exit 0
  fi
  if [ "$waited" -ge "$timeout" ]; then
    echo "::error::npm still does not serve $pkg@$version after ${waited}s. Check that the npm-publish job published it, then rerun this job."
    exit 1
  fi
  echo "npm does not serve $pkg@$version yet (waited ${waited}s of ${timeout}s)"
  sleep "$interval"
  waited=$((waited + interval))
done
