# linkwarden-obsidian-sync

[![ci](https://github.com/alrayyes/linkwarden-obsidian-sync/actions/workflows/ci.yml/badge.svg)](https://github.com/alrayyes/linkwarden-obsidian-sync/actions/workflows/ci.yml)
[![Codecov](https://codecov.io/gh/alrayyes/linkwarden-obsidian-sync/graph/badge.svg)](https://codecov.io/gh/alrayyes/linkwarden-obsidian-sync)
[![release](https://img.shields.io/github/v/release/alrayyes/linkwarden-obsidian-sync)](https://github.com/alrayyes/linkwarden-obsidian-sync/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/alrayyes/linkwarden-obsidian-sync.svg)](https://pkg.go.dev/github.com/alrayyes/linkwarden-obsidian-sync)
[![license](https://img.shields.io/github/license/alrayyes/linkwarden-obsidian-sync)](LICENSE)

Copies newly saved [Linkwarden](https://linkwarden.app) links into an Obsidian
vault as notes, then pushes them to a dated branch for review. No LLM in the
loop — everything here is a deterministic API call, file write and git
command, meant to run unattended on a schedule (a `systemd --user` timer).

## What it does

1. Reads the last-synced marker at `$XDG_STATE_HOME/linkwarden-obsidian-sync/last-synced.json`
   (default `~/.local/state/linkwarden-obsidian-sync/`).
2. Pages through Linkwarden's `/api/v1/search` endpoint, newest-first, until
   it reaches a link already synced. Two links saved in the same second are
   disambiguated by ID, not just by timestamp — Linkwarden's `createdAt` only
   has second resolution.
3. Writes each new link into `<vault_path>/<vault_subdir>/<slug>.md`
   (default `Linkwarden/`) as a note with YAML frontmatter (`url`,
   `collection`, `tags`, `created`, `linkwarden_id`) and the link's own
   description as the body. Never overwrites a note that already exists on
   disk — a rerun after a partial failure won't clobber something you've
   started editing.
4. If anything was written, commits on a branch named
   `linkwarden-sync-<timestamp>` and pushes it. Prints the compare URL; it
   does **not** open the pull request itself — that's still a human step.

## Configuration

Settings live in a TOML file at `$XDG_CONFIG_HOME/linkwarden-obsidian-sync/config.toml`
(falling back to `~/.config/linkwarden-obsidian-sync/config.toml` per the XDG
Base Directory spec), or wherever `--config` points instead. Generate one to
start from:

```shell
linkwarden-obsidian-sync init
```

which writes a commented template — including every default — to the
resolved path and refuses to overwrite an existing file unless you pass
`--force`. Running the bare command with nothing configured yet offers to
do the same thing on the spot — answer yes, or pass `--yes` up front to
skip asking; with no terminal to ask on (a script, a systemd unit), it
skips the prompt and fails with the same error `init` points you at.
Fill in the two required values and you're done:

```toml
linkwarden_url = "https://linkwarden.example.com"
linkwarden_token = "eyJ..."
```

| Key                | Required | Default                                          |
| ------------------ | -------- | ------------------------------------------------- |
| `linkwarden_url`   | yes      | —                                                   |
| `linkwarden_token` | yes      | —                                                   |
| `vault_path`       | no       | `~/Documents/obsidian`                              |
| `vault_subdir`     | no       | `Linkwarden`                                        |
| `state_dir`        | no       | `$XDG_STATE_HOME/linkwarden-obsidian-sync`          |
| `skip_git`         | no       | `false`                                             |

Every key can also be set via the environment variable of the same shape —
`LINKWARDEN_URL`, `LINKWARDEN_TOKEN`, `VAULT_PATH`, `VAULT_SUBDIR`,
`LINKWARDEN_SYNC_STATE_DIR`, `LINKWARDEN_SYNC_SKIP_GIT` — which **overrides**
the config file when both are set, useful for configuring this without a
mounted file (a container, say). `--skip-git` also works as a one-off flag,
regardless of what's in the file.

`linkwarden_token` is a JWT from Linkwarden's own **Settings → Access
Tokens**, sent as `Authorization: Bearer <token>`. It's a personal credential
scoped to your Linkwarden account, not something this tool generates — keep
the config file's permissions tight (`init` writes it `0600`) since it holds
a secret in plain text, same as any env file would.

`vault_path` needs to already be a git clone with working push credentials
configured (the same clone you use interactively is fine) — this tool shells
out to the `git` on `PATH`, it doesn't carry its own.

## Running it

Grab a prebuilt binary from the [latest
release](https://github.com/alrayyes/linkwarden-obsidian-sync/releases/latest)
(linux/darwin, amd64/arm64), install it with Go:

```shell
go install github.com/alrayyes/linkwarden-obsidian-sync/cmd/linkwarden-obsidian-sync@latest
```

or build it from a clone:

```shell
go build -o linkwarden-obsidian-sync ./cmd/linkwarden-obsidian-sync
```

Then, after `linkwarden-obsidian-sync init` and filling in the config file:

```shell
linkwarden-obsidian-sync
```

Exits non-zero and logs the reason on any failure — a systemd unit running
this on a timer should treat a non-zero exit as worth looking at.

## Testing

```shell
go build ./... && go vet ./... && go test ./... -race -cover
go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for testing against a real
Linkwarden instance via `compose.yaml`.
