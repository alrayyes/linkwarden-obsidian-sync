#!/usr/bin/env sh
# Tests scripts/update-vendor-hash.sh against a stub `nix`, so no Nix install
# is needed. Run by ci.yml's `test` job.
set -eu

here="$(cd "$(dirname "$0")" && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail=0

OLD='sha256-3QYTRBLglXGUGIg5O7iWHLGFmAc+aEXsw2P4wzyZkh0='
NEW='sha256-1Ma1at3Ladp8BkPLSX4Yypp37cR6O8/962BaTox7bP4='

# stub <mode>: writes a fake nix that behaves per mode.
stub() {
  cat >"$tmp/nix" <<STUB
#!/usr/bin/env sh
case "$1" in
  ok) exit 0 ;;
  stale)
    # Fails while flake.nix still holds the old hash, passes once it's fixed.
    if grep -q "$NEW" flake.nix; then exit 0; fi
    echo "error: hash mismatch in fixed-output derivation '/nix/store/x.drv':"
    echo "         specified: $OLD"
    echo "            got:    $NEW"
    exit 1 ;;
  noisy)
    # -L output: a build step printing its own "got:" must not be mistaken
    # for Nix's report, and a second mismatch means a guess, not a fix.
    echo "linkwarden-obsidian-sync> got: sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
    echo "error: hash mismatch in fixed-output derivation '/nix/store/x.drv':"
    echo "         specified: $OLD"
    echo "            got:    $NEW"
    echo "error: hash mismatch in fixed-output derivation '/nix/store/y.drv':"
    echo "         specified: $OLD"
    echo "            got:    sha256-BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB="
    exit 1 ;;
  prefixed)
    echo "linkwarden-obsidian-sync> got: $NEW"
    echo "error: builder failed"
    exit 1 ;;
  other)
    echo "error: builder for '/nix/store/x.drv' failed with exit code 1"
    exit 1 ;;
esac
STUB
  chmod +x "$tmp/nix"
}

# run <mode>: fresh flake.nix, runs the script, sets $status and $out.
run() {
  stub "$1"
  printf '  vendorHash = "%s";\n  other = "x";\n' "$OLD" >"$tmp/flake.nix"
  status=0
  out="$(cd "$tmp" && NIX="$tmp/nix" sh "$here/update-vendor-hash.sh" 2>&1)" || status=$?
}

check() { # <description> <condition-exit-code>
  if [ "$2" -ne 0 ]; then
    echo "FAIL: $1" >&2
    echo "  output: $out" >&2
    fail=1
  fi
}

run ok
check "a passing build exits 0" "$status"
grep -q "$OLD" "$tmp/flake.nix"; check "a passing build leaves flake.nix alone" $?

run stale
check "a stale hash is fixed, exit 0" "$status"
grep -q "vendorHash = \"$NEW\";" "$tmp/flake.nix"; check "the stale hash is replaced with the one nix reported" $?
grep -q 'other = "x";' "$tmp/flake.nix"; check "other lines are untouched" $?

run other
[ "$status" -ne 0 ]; check "a non-hash build failure still fails" $?
grep -q "$OLD" "$tmp/flake.nix"; check "a non-hash failure leaves flake.nix alone" $?

run noisy
[ "$status" -ne 0 ]; check "two different mismatches fail rather than guess" $?
grep -q "$OLD" "$tmp/flake.nix"; check "two mismatches leave flake.nix alone" $?

run prefixed
[ "$status" -ne 0 ]; check "a prefixed build-log 'got:' is not Nix's report" $?
grep -q "$OLD" "$tmp/flake.nix"; check "a prefixed 'got:' leaves flake.nix alone" $?

exit "$fail"
