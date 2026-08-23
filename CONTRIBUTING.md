# Contributing

A single small Go CLI, no subpackages.

- `linkwarden.go` — the Linkwarden API client and pagination/stop logic.
- `note.go` — turning a link into a markdown note, and writing it safely.
- `state.go` — the on-disk last-synced marker.
- `main.go` — wiring, config, and the git commit/push step.

## Running tests

```shell
go build ./... && go vet ./... && gofmt -l . && go test ./... -race -cover
```

`gofmt -l .` should print nothing; a non-empty result means a file isn't
formatted. CI runs the same four commands.

## Adding a change

Small, atomic commits — one logical change each, building and passing tests
on its own. [Conventional Commits](https://www.conventionalcommits.org/) for
the message. Work lands through a pull request against `main`.
