# Contributing

- `cmd/linkwarden-obsidian-sync/main.go` — the composition root: wiring,
  config, and the top-level sync loop.
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

## Adding a change

Small, atomic commits — one logical change each, building and passing tests
on its own. [Conventional Commits](https://www.conventionalcommits.org/) for
the message. Work lands through a pull request against `main`.
