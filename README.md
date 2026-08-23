# linkwarden-obsidian-sync

Copies newly saved [Linkwarden](https://linkwarden.app) links into an Obsidian
vault as notes, then pushes them to a dated branch for review. No LLM in the
loop — everything here is a deterministic API call, file write and git
command, meant to run unattended on a schedule (a `systemd --user` timer).

## What it does

1. Reads the last-synced marker at `$LINKWARDEN_SYNC_STATE_DIR/last-synced.json`
   (default `~/.local/state/linkwarden-obsidian-sync/`).
2. Pages through Linkwarden's `/api/v1/search` endpoint, newest-first, until
   it reaches a link already synced. Two links saved in the same second are
   disambiguated by ID, not just by timestamp — Linkwarden's `createdAt` only
   has second resolution.
3. Writes each new link into `$VAULT_PATH/$VAULT_SUBDIR/<slug>.md`
   (default `Linkwarden/`) as a note with YAML frontmatter (`url`,
   `collection`, `tags`, `created`, `linkwarden_id`) and the link's own
   description as the body. Never overwrites a note that already exists on
   disk — a rerun after a partial failure won't clobber something you've
   started editing.
4. If anything was written, commits on a branch named
   `linkwarden-sync-<timestamp>` and pushes it. Prints the compare URL; it
   does **not** open the pull request itself — that's still a human step.

## Configuration

| Variable                   | Required | Default                                     |
| --------------------------- | -------- | ---------------------------------------------- |
| `LINKWARDEN_URL`            | yes      | —                                              |
| `LINKWARDEN_TOKEN`          | yes      | —                                              |
| `VAULT_PATH`                | no       | `~/Documents/obsidian`                         |
| `VAULT_SUBDIR`              | no       | `Linkwarden`                                   |
| `LINKWARDEN_SYNC_STATE_DIR` | no       | `~/.local/state/linkwarden-obsidian-sync`      |
| `LINKWARDEN_SYNC_SKIP_GIT`  | no       | unset — set to anything to skip the git commit |

`LINKWARDEN_TOKEN` is a JWT from Linkwarden's own **Settings → Access
Tokens**, sent as `Authorization: Bearer <token>`. It's a personal credential
scoped to your Linkwarden account, not something this tool generates.

`VAULT_PATH` needs to already be a git clone with working push credentials
configured (the same clone you use interactively is fine) — this tool shells
out to the `git` on `PATH`, it doesn't carry its own.

## Running it

```shell
go build -o linkwarden-obsidian-sync .
LINKWARDEN_URL=https://linkwarden.example.com \
LINKWARDEN_TOKEN=eyJ... \
./linkwarden-obsidian-sync
```

Exits non-zero and logs the reason on any failure — a systemd unit running
this on a timer should treat a non-zero exit as worth looking at.

## Testing

```shell
go build ./... && go vet ./... && go test ./... -race -cover
```
