#!/usr/bin/env bash
# Assembles the test and coverage reports the CI `reports` job publishes at
# apis.ryankes.eu/linkwarden-obsidian-sync/reports/.
#
# usage: build-reports.sh <site-dir> <junit.xml> <coverage.out>
#
# Writes <site-dir>/reports/. The Pages root for this repo is already
# /linkwarden-obsidian-sync/, so the layout starts at reports/ with no repo
# prefix. REPORTS_COMMIT and REPORTS_DATE override the values shown on the
# index page (the tests use them to stay deterministic).
set -euo pipefail

site="${1:?usage: build-reports.sh <site-dir> <junit.xml> <coverage.out>}"
junit="${2:?missing junit.xml}"
profile="${3:?missing coverage.out}"

commit="${REPORTS_COMMIT:-$(git rev-parse HEAD)}"
date="${REPORTS_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
reports="$site/reports"

mkdir -p "$reports/tests" "$reports/coverage"

cp "$junit" "$reports/tests/unit.xml"
cp "$profile" "$reports/coverage/coverage.out"

# Cobertura is the one coverage format every viewer reads. The version is
# pinned, and `go run pkg@version` keeps it out of go.mod so `go mod tidy
# -diff` stays clean.
go run github.com/boumenot/gocover-cobertura@v1.5.0 < "$profile" > "$reports/coverage/coverage.xml"
go tool cover -html="$profile" -o "$reports/coverage/index.html"

cat > "$reports/index.html" <<HTML
<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>linkwarden-obsidian-sync reports</title>
</head>
<body>
<main>
<h1>linkwarden-obsidian-sync reports</h1>
<p>Commit <code>${commit:0:7}</code>, published ${date}.</p>
<ul>
<li><a href="tests/unit.xml">Test results</a> (JUnit XML from gotestsum)</li>
<li><a href="coverage/">Coverage</a> (HTML view)
 <ul>
 <li><a href="coverage/coverage.xml">coverage.xml</a> (Cobertura)</li>
 <li><a href="coverage/coverage.out">coverage.out</a> (Go profile)</li>
 </ul>
</li>
</ul>
</main>
</body>
</html>
HTML
