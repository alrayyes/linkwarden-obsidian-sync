# Contributing

- `cmd/linkwarden-obsidian-sync/main.go` — the composition root: the cobra
  command tree (`init`, and the root command's sync) and the top-level sync
  loop.
- `internal/config` — loading and validating the TOML config file, its
  environment-variable overrides, and writing `init`'s template.
- `internal/linkwarden` — the Linkwarden API client and pagination/stop
  logic.
- `internal/note` — turning a link into a markdown note, and writing it
  safely.
- `internal/state` — the on-disk last-synced marker.
- `internal/vault` — committing and pushing new notes into the Obsidian
  vault checkout.

## Running tests

```shell
go build ./... && go vet ./... && gofmt -l . && go mod tidy -diff && go test ./... -race -cover
```

`gofmt -l .` should print nothing; a non-empty result means a file isn't
formatted. `go mod tidy -diff` should print nothing too; a non-empty result
means `go.mod`/`go.sum` have drifted from what `go mod tidy` would produce.

Lint with golangci-lint, configured in `.golangci.yml`:

```shell
golangci-lint run ./...
```

CI runs the same checks, plus `govulncheck` for known CVEs in pinned
dependencies.

## Getting set up

- **Go**, version matching the `go` directive in `go.mod`.
- **[bun](https://bun.sh)**, for the tooling that isn't Go — commitlint and
  the [lefthook](https://lefthook.dev) that runs the git hooks. There's a
  `package.json`, but nothing here is JavaScript; it exists only so those
  tools resolve and stay pinned.
- **[Docker](https://docker.com)**, which the hooks use to run `gofmt`,
  `golangci-lint` and `go test` at a pinned version rather than whatever the
  host toolchain happens to have.

One command installs the tooling and the git hooks:

```shell
bun install
```

An uninstalled hook silently does nothing, which is worse than not having
one, so the `prepare` script runs `lefthook install` for you. You find out
at the pipeline otherwise, not at the commit.

## Adding a change

Small, atomic commits — one logical change each, building and passing tests
on its own. [Conventional Commits](https://www.conventionalcommits.org/) for
the message: `type(scope): description`, types `feat`/`fix`/`docs`/`style`/
`refactor`/`perf`/`test`/`build`/`ci`/`chore`/`revert`. Subject under 50
characters, lowercase, no trailing full stop. commitlint enforces the shape
at `commit-msg` and again in CI; the length and case rules are tighter than
what it checks, so hold to them anyway.

Work lands through a pull request against `main`. The pull request **title**
has to be a valid Conventional Commit too — `pr-title.yml` checks it.
commitlint only ever reads commit objects, and this repo merges via squash,
which defaults the squash commit's message to the pull request title — so
the title check is the only thing standing between a badly titled pull
request and a bad message on `main`.
