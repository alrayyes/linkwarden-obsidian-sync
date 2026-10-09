#!/usr/bin/env bash
# Checks that build-reports.sh lays the reports out the way the Pages job and
# the live-URL check expect. Run from the repository root.
set -euo pipefail

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

cat > "$work/junit.xml" <<'XML'
<?xml version="1.0" encoding="UTF-8"?>
<testsuites><testsuite name="x" tests="1" failures="0"><testcase name="TestX"/></testsuite></testsuites>
XML

go test ./internal/config -coverprofile="$work/coverage.out" > /dev/null

site="$work/site"
REPORTS_COMMIT=0123456789abcdef REPORTS_DATE=2026-01-02T03:04:05Z \
  ./scripts/build-reports.sh "$site" "$work/junit.xml" "$work/coverage.out"

fail() {
  echo "FAIL: $1" >&2
  exit 1
}

for f in \
  reports/index.html \
  reports/tests/unit.xml \
  reports/coverage/index.html \
  reports/coverage/coverage.xml \
  reports/coverage/coverage.out; do
  [ -s "$site/$f" ] || fail "$f is missing or empty"
done

grep -q '<coverage ' "$site/reports/coverage/coverage.xml" || fail "coverage.xml is not Cobertura"
grep -q '<testsuite' "$site/reports/tests/unit.xml" || fail "unit.xml is not JUnit"
grep -q '0123456' "$site/reports/index.html" || fail "index.html lacks the commit"
grep -q '2026-01-02' "$site/reports/index.html" || fail "index.html lacks the date"
for link in tests/unit.xml coverage/ coverage/coverage.xml coverage/coverage.out; do
  grep -q "href=\"$link\"" "$site/reports/index.html" || fail "index.html does not link $link"
done

echo "build-reports: ok"
