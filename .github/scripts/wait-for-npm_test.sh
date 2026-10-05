#!/usr/bin/env bash
# Tests wait-for-npm.sh against a fake curl on PATH, so it needs no network.
# The fake serves only the version document the MCP Registry fetches,
# https://registry.npmjs.org/<pkg>/<version>, and only once
# FAKE_CURL_READY_AFTER calls have answered 404 (curl -f exits 22). Any other
# URL, the package document included, answers 404 forever.
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

cat > "$tmp/curl" <<'FAKE'
#!/usr/bin/env bash
n=$(( $(cat "$FAKE_CURL_CALLS" 2>/dev/null || echo 0) + 1 ))
echo "$n" > "$FAKE_CURL_CALLS"
url="${*: -1}"
if [ "$url" = "https://registry.npmjs.org/genieacs-mcp/1.2.3" ] && [ "$n" -gt "$FAKE_CURL_READY_AFTER" ]; then
  echo '{"name":"genieacs-mcp","version":"1.2.3","mcpName":"io.github.GeiserX/genieacs-mcp"}'
  exit 0
fi
echo "curl: (22) The requested URL returned error: 404" >&2
exit 22
FAKE
chmod +x "$tmp/curl"

fail=0
run() { # name, ready-after, want-exit, want-calls, want-output, [args]
  local name="$1" ready="$2" want_exit="$3" want_calls="$4" want_out="$5" code=0
  shift 5
  [ $# -gt 0 ] || set -- genieacs-mcp 1.2.3
  rm -f "$tmp/calls"
  PATH="$tmp:$PATH" FAKE_CURL_CALLS="$tmp/calls" FAKE_CURL_READY_AFTER="$ready" \
    WAIT_TIMEOUT=2 WAIT_INTERVAL=1 \
    "$here/wait-for-npm.sh" "$@" > "$tmp/out" 2>&1 || code=$?
  local calls; calls=$(cat "$tmp/calls" 2>/dev/null || echo 0)
  if [ "$code" != "$want_exit" ] || [ "$calls" != "$want_calls" ] || ! grep -qF -- "$want_out" "$tmp/out"; then
    echo "FAIL $name: exit $code (want $want_exit), curl calls $calls (want $want_calls), want output containing: $want_out"
    sed 's/^/  | /' "$tmp/out"
    fail=1
  else
    echo "ok   $name"
  fi
}

served="npm serves genieacs-mcp@1.2.3"
gave_up="::error::npm still does not serve genieacs-mcp@1.2.3"
usage="::error::usage: wait-for-npm.sh"

run "served at once"                   0  0 1 "$served"
run "served on the 3rd check"          2  0 3 "$served"
run "never served, times out"          99 1 3 "$gave_up"
run "null version from jq, no polling" 0  2 0 "$usage" genieacs-mcp null
run "missing version, no polling"      0  2 0 "$usage" genieacs-mcp

exit "$fail"
