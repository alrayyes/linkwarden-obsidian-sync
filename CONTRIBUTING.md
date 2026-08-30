# Contributing

- `cmd/linkwarden-obsidian-sync/main.go` — the composition root: the cobra
  command tree (`init`, and the root command's sync) and the top-level sync
  loop.
- `internal/config` — loading and validating the TOML config file, its
  environment-variable overrides, and writing `init`'s template.
- `internal/linkwarden` — the Linkwarden API client and pagination logic.
- `internal/note` — turning a link into a markdown note, writing it, and
  removing it.
- `internal/state` — the on-disk marker of every currently-known link and
  its note's filename.
- `internal/reconcile` — the add/update/remove diff between what
  Linkwarden has now and what the vault had last run.

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
dependencies, and separately lints and builds the `Dockerfile`:

```shell
docker run --rm -i hadolint/hadolint:v2.15.1 < Dockerfile
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o linkwarden-obsidian-sync ./cmd/linkwarden-obsidian-sync
docker build -t linkwarden-obsidian-sync:local .
```

hadolint reads the Dockerfile as text; the actual `docker build` is what
proves it still works, which hadolint alone never checks.

## Getting set up

- **Go**, version matching the `go` directive in `go.mod`.
- **[bun](https://bun.sh)**, for the tooling that isn't Go — commitlint and
  the [lefthook](https://lefthook.dev) that runs the git hooks. There's a
  `package.json`, but nothing here is JavaScript; it exists only so those
  tools resolve and stay pinned.
- **[Docker](https://docker.com)**, which the hooks use to run `gofmt`,
  `golangci-lint` and `go test` at a pinned version rather than whatever the
  host toolchain happens to have, and to build/lint the `Dockerfile` itself.

One command installs the tooling and the git hooks:

```shell
bun install
```

An uninstalled hook silently does nothing, which is worse than not having
one, so the `prepare` script runs `lefthook install` for you. You find out
at the pipeline otherwise, not at the commit.

## Testing against a real Linkwarden instance

`internal/linkwarden`'s tests run against an `httptest` fake modeling
Linkwarden's actual `/api/v1/search` contract — but a fake can drift from
the real thing (it happened once already: the response shape and cursor
pagination scheme were both wrong until a real instance caught it). To
check the fake — or a suspected API-shape bug — against reality:

```shell
docker compose up -d
```

brings up Postgres and Linkwarden (`compose.yaml`; deliberately no
Meilisearch — see the file's own comment for why). Once
`curl -s http://localhost:3000/api/v1/auth/csrf` returns `200`, register a
user, log in through NextAuth's credentials flow, and mint an access
token:

```shell
curl -s -X POST http://localhost:3000/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Test User","username":"testuser","password":"testpassword123","email":""}'

COOKIES=$(mktemp)
CSRF=$(curl -s -c "$COOKIES" http://localhost:3000/api/v1/auth/csrf | jq -r .csrfToken)
curl -s -b "$COOKIES" -c "$COOKIES" -X POST http://localhost:3000/api/v1/auth/callback/credentials \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data-urlencode "username=testuser" \
  --data-urlencode "password=testpassword123" \
  --data-urlencode "csrfToken=$CSRF" \
  --data-urlencode "json=true"

TOKEN=$(curl -s -b "$COOKIES" -X POST http://localhost:3000/api/v1/tokens \
  -H "Content-Type: application/json" \
  -d '{"name":"sync-test","expires":4}' | jq -r .response.secretKey)
```

(`expires: 4` is Linkwarden's `TokenExpiry.never` — see
`packages/types/global.ts` upstream; the API takes that enum, not a day
count.) Save a link or two through the UI at <http://localhost:3000>, then
point the binary at it:

```shell
LINKWARDEN_URL=http://localhost:3000 LINKWARDEN_TOKEN="$TOKEN" \
VAULT_PATH=/tmp/test-vault \
go run ./cmd/linkwarden-obsidian-sync
```

Rename or delete a link through the UI and run it again to exercise the
update/rename/delete paths, not just add — that's most of what
`internal/reconcile` actually does.

`docker compose down -v` tears it down. This isn't wired into CI — minting
a token needs the full login flow above, not a single request a workflow
step could make on its own — so it's a manual check, reached for when a
fake's fidelity to the real API is actually in question.

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
