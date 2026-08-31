# linkwarden-obsidian-sync

[![ci](https://github.com/alrayyes/linkwarden-obsidian-sync/actions/workflows/ci.yml/badge.svg)](https://github.com/alrayyes/linkwarden-obsidian-sync/actions/workflows/ci.yml)
[![Codecov](https://codecov.io/gh/alrayyes/linkwarden-obsidian-sync/graph/badge.svg)](https://codecov.io/gh/alrayyes/linkwarden-obsidian-sync)
[![release](https://img.shields.io/github/v/release/alrayyes/linkwarden-obsidian-sync)](https://github.com/alrayyes/linkwarden-obsidian-sync/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/alrayyes/linkwarden-obsidian-sync.svg)](https://pkg.go.dev/github.com/alrayyes/linkwarden-obsidian-sync)
[![license](https://img.shields.io/github/license/alrayyes/linkwarden-obsidian-sync)](LICENSE)

Keeps a directory of Obsidian notes in sync with your saved
[Linkwarden](https://linkwarden.app) links: one note per link, added,
updated or removed to match Linkwarden's current state exactly. No LLM in
the loop — everything here is a deterministic API call and a file write,
meant to run unattended on a schedule (a `systemd --user` timer). It
doesn't touch git; reviewing and committing whatever changed is your own
workflow from here.

## Quick start

```shell
go install github.com/alrayyes/linkwarden-obsidian-sync/cmd/linkwarden-obsidian-sync@latest
linkwarden-obsidian-sync init   # writes a config template and tells you where
# edit it: set linkwarden_url and linkwarden_token
linkwarden-obsidian-sync
```

A prebuilt binary (linux/darwin, amd64/arm64) is on the [latest
release](https://github.com/alrayyes/linkwarden-obsidian-sync/releases/latest)
if you'd rather not build with Go. Debian/Ubuntu and Fedora/RHEL users can
grab a `.deb` or `.rpm` from the same release instead:

```shell
# Debian/Ubuntu
sudo dpkg -i linkwarden-obsidian-sync_*_linux_amd64.deb

# Fedora/RHEL
sudo rpm -i linkwarden-obsidian-sync_*_linux_amd64.rpm
```

or run it as a container:

```shell
docker run --rm \
  -e LINKWARDEN_URL=https://linkwarden.example.com \
  -e LINKWARDEN_TOKEN=eyJ... \
  -e VAULT_PATH=/vault -v ~/Documents/obsidian:/vault \
  -e LINKWARDEN_SYNC_STATE_DIR=/state -v linkwarden-sync-state:/state \
  ghcr.io/alrayyes/linkwarden-obsidian-sync:latest
```

(mount `/state` too, or every run treats every link as new — see
[docs/USAGE.md](docs/USAGE.md#docker))

See **[docs/USAGE.md](docs/USAGE.md)** for exactly what gets added, updated
and removed on a run (with examples), the full configuration reference, and
the state file's shape.

## Testing

```shell
go build ./... && go vet ./... && go test ./... -race -cover
go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for testing against a real
Linkwarden instance via `compose.yaml`.
