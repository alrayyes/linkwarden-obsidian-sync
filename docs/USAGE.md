# Usage

## What a run actually does

Every run fetches your **complete** current set of Linkwarden links — not
just what's new — and makes `<vault_path>/<vault_subdir>/` match it
exactly:

- A link with no note yet gets one, **added**.
- A link whose note already exists gets its content **overwritten** to
  match Linkwarden's current title, tags, collection and description —
  even over a hand-edit. This tool doesn't diff content first; if you want
  to keep something you wrote in a note, keep it somewhere else.
- A link whose name changed gets **renamed**: the old file (whatever slug
  its old name produced) is removed, the new one written.
- A note whose link no longer exists in Linkwarden — you deleted or
  unsaved it — gets **removed**.

Fetching everything each run (rather than only what's new) is what makes
deletion detection possible at all; there's no "what changed since X" query
this API offers instead. For a few hundred links this is a handful of
paginated requests and takes well under a second.

Nothing here touches git. Reviewing the resulting diff — `git status`,
`git diff`, deciding what to commit — is your own workflow from here, the
same as any other change you'd make to the vault by hand.

## Example run

Starting from two saved links and an empty vault:

```console
$ linkwarden-obsidian-sync
2026/08/30 linkwarden-obsidian-sync 1.3.0
2026/08/30 synced 2 link(s): 2 added, 0 updated, 0 removed
```

```console
$ find ~/Documents/obsidian/Linkwarden -type f
~/Documents/obsidian/Linkwarden/Go by Example.md
~/Documents/obsidian/Linkwarden/Effective Go.md
```

Rename one of those links in Linkwarden, delete the other, and save a new
one, then run again:

```console
$ linkwarden-obsidian-sync
2026/08/30 linkwarden-obsidian-sync 1.3.0
2026/08/30 synced 2 link(s): 1 added, 1 updated, 1 removed
```

```console
$ find ~/Documents/obsidian/Linkwarden -type f
~/Documents/obsidian/Linkwarden/Go by Example, Annotated.md
~/Documents/obsidian/Linkwarden/A New Article.md
```

`Effective Go.md` is gone (its link was deleted); `Go by Example.md` was
renamed to `Go by Example, Annotated.md` (the link's title changed); `A New
Article.md` is the newly saved link.

## Example note

A saved link renders as:

```markdown
---
url: "https://go.dev/doc/effective_go"
collection: "Reading"
tags:
  - "go"
  - "reference"
created: 2026-08-30
linkwarden_id: 42
---

# Effective Go

<https://go.dev/doc/effective_go>

Tips for writing clear, idiomatic Go code.
```

The frontmatter is there to query on in Obsidian (`tags`, `collection`),
and `linkwarden_id` is what this tool itself uses to recognize the note as
"already synced" across runs — don't hand-edit it.

## Configuration reference

Settings live in a TOML file at
`$XDG_CONFIG_HOME/linkwarden-obsidian-sync/config.toml` (falling back to
`~/.config/linkwarden-obsidian-sync/config.toml` per the XDG Base
Directory spec), or wherever `--config` points instead.
`linkwarden-obsidian-sync init` writes a commented template there,
including every default, and refuses to overwrite an existing file unless
you pass `--force`.

A filled-in example:

```toml
linkwarden_url = "https://linkwarden.example.com"
linkwarden_token = "eyJhbGciOiJIUzI1NiJ9.eyJpZCI6MX0.xxxxxxxxxxxxxxxxxxxx"
vault_path = "/home/alrayyes/Documents/obsidian"
vault_subdir = "Linkwarden"
state_dir = "/home/alrayyes/.local/state/linkwarden-obsidian-sync"
```

| Key                | Required | Default                                    |
| ------------------ | -------- | ------------------------------------------ |
| `linkwarden_url`   | yes      | —                                          |
| `linkwarden_token` | yes      | —                                          |
| `vault_path`       | no       | `~/Documents/obsidian`                     |
| `vault_subdir`     | no       | `Linkwarden`                               |
| `state_dir`        | no       | `$XDG_STATE_HOME/linkwarden-obsidian-sync` |

Every key can also be set via the environment variable of the same shape —
`LINKWARDEN_URL`, `LINKWARDEN_TOKEN`, `VAULT_PATH`, `VAULT_SUBDIR`,
`LINKWARDEN_SYNC_STATE_DIR` — which **overrides** the config file when both
are set, useful for configuring this without a mounted file (a container,
say).

`linkwarden_token` is a JWT from Linkwarden's own **Settings → Access
Tokens**, sent as `Authorization: Bearer <token>`. It's a personal
credential scoped to your Linkwarden account, not something this tool
generates — keep the config file's permissions tight (`init` writes it
`0600`) since it holds a secret in plain text, same as any env file would.

Running the bare command with nothing configured yet — no config file, no
`LINKWARDEN_URL`/`LINKWARDEN_TOKEN` — offers to write the template on the
spot: answer yes, or pass `--yes` up front to skip asking. With no
terminal to ask on (a script, a systemd unit), it skips the prompt and
fails with the same error `init` points you at, rather than hanging.

## State

`state_dir/last-synced.json` maps every currently known link's ID to its
note's current filename:

```json
{
  "links": {
    "42": "Effective Go.md",
    "51": "Go by Example.md"
  }
}
```

This is what makes a rename and a deletion detectable — the ID is stable,
the filename isn't. Delete this file to force a full add-everything resync
(safe: every note gets overwritten with the same content it already has,
nothing is deleted since there's no "previous" state to diff against).

## Docker

```shell
docker run --rm \
  -e LINKWARDEN_URL=https://linkwarden.example.com \
  -e LINKWARDEN_TOKEN=eyJ... \
  -e VAULT_PATH=/vault -v ~/Documents/obsidian:/vault \
  -e LINKWARDEN_SYNC_STATE_DIR=/state -v linkwarden-sync-state:/state \
  ghcr.io/alrayyes/linkwarden-obsidian-sync:latest
```

Configuration is entirely through environment variables here — see the
[reference above](#configuration-reference) for the full list — since
there's no config file mounted in by default. `VAULT_SUBDIR` defaults to
`Linkwarden` same as anywhere else, so it's usually not worth setting.

**Mount `state_dir` to a persistent volume.** Without it, the container
starts from empty state on every run: every link looks "new" again (an
already-existing note just gets rewritten with the same content — no
duplication, no data loss, just wasted work) and a deletion in Linkwarden
is never detected, since there's no previous state to diff against. A
named volume (`linkwarden-sync-state` above) or a bind mount both work.

To run it on a schedule, put the `docker run` command in a `cron`/systemd
timer the same way you would the bare binary — the image doesn't include
its own scheduler.

Images are published for `linux/amd64` and `linux/arm64`, signed with a
build provenance attestation (`gh attestation verify` against
`ghcr.io/alrayyes/linkwarden-obsidian-sync`).
