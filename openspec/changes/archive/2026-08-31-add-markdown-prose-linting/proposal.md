## Why

This repo lints code but not prose: nothing catches a broken heading, a
grammar mistake, or an off-voice sentence in README.md, CONTRIBUTING.md,
SECURITY.md, docs/USAGE.md, or docs/INSTALL.md before it merges. A rules
audit (tracked in GitHub issue #45) found the full four-tool pipeline
missing entirely — no Prettier, markdownlint-cli2, Vale, or ltex-cli-plus,
and nothing wired into `lefthook.yml` or `ci.yml`.

Two spikes against this repo's actual docs (not a generic estimate)
found the real cost isn't wiring the four tools — it's two decisions
that have to be made before any of them are usable:

- **ltex-cli-plus exits 3 on any finding, `info` severity included.**
  Run cold against just README.md, it already fails on "Linkwarden" and
  "Codecov" — the project's own name. Without a vocabulary list, every
  push touching docs fails immediately.
- **Vale's Google style, applied as-is, contradicts this repo's own
  established voice.** ~40 of 77 findings across the five docs are
  `Google.EmDash` flagging the spaced em-dashes CLAUDE.md's own
  "Writing for a reader" section mandates ("Go easy on em-dashes. One
  where a sentence genuinely turns.") — confirmed by grep, this repo's
  prose uses them throughout on purpose. `Google.Headings` and
  `Google.WordListCase` raise similar fit questions (title-case section
  headings throughout, and "touch" flagged as UI language on a line
  about the Unix command). These need explicit per-rule decisions, not
  a vocabulary fix.

## What Changes

- Add and configure four prose-linting tools: **Prettier** (layout),
  **markdownlint-cli2** (structure), **Vale** (style), **ltex-cli-plus**
  (grammar/spelling).
- Build a House vocabulary covering this repo's actual jargon — spiked
  at 25+ entries from Vale alone across the five current docs
  (Linkwarden, Codecov, hadolint, Dockerfile, commitlint, lefthook,
  Postgres, Meilisearch, toolchain, frontmatter, systemd, nixpkgs,
  resync, config, env, repo, enum, CVEs, and more), reused by both Vale
  and LTeX per their own separate dictionary formats.
- Decide and document which Google style sub-rules apply to this repo's
  voice — `Google.EmDash` off (contradicts the mandated spaced-em-dash
  style), `Google.Headings` and `Google.WordListCase` evaluated case by
  case rather than accepted wholesale.
- Wire all four into `lefthook.yml`: Prettier's fixer + markdownlint +
  Vale in `pre-commit` (all fast enough), LTeX in `pre-push` only (its
  ~300MB one-time download makes it too heavy for `pre-commit`, though
  a cached run over this repo's five docs takes ~8 seconds regardless
  of file count, confirmed live).
- Wire Vale and LTeX into `ci.yml` as two separate jobs, so a red
  pipeline names which tier failed.
- Get README.md, CONTRIBUTING.md, SECURITY.md, docs/USAGE.md, and
  docs/INSTALL.md passing all four tools clean, with no suppressions
  beyond the House vocabulary.
- Document the new tooling in CONTRIBUTING.md's "Getting set up"
  section, alongside the existing Go/bun/Docker requirements.

## Capabilities

### New Capabilities

- `docs-linting`: every tracked Markdown doc in this repo passes
  layout, structure, mechanics (grammar/spelling), and style checks in
  both local hooks and CI before it can merge — mechanics fail the
  build, style warns.

### Modified Capabilities

(none — no existing specs cover documentation tooling)

## Impact

- **New config files** at the repo root: `.prettierrc` (or its
  `overrides`), `.markdownlint-cli2.yaml`, `.vale.ini`,
  `styles/config/vocabularies/House/accept.txt`, an LTeX
  client-configuration JSON.
- **`lefthook.yml`**: new `pre-commit` jobs (Prettier fix, markdownlint,
  Vale) and a new `pre-push` job (LTeX), scoped to Markdown (and, for
  Prettier, the YAML files it already should cover per `markdown.md`).
- **`.github/workflows/ci.yml`**: two new jobs (`vale`, `ltex`), each
  running its own setup (`vale sync`; LTeX's release tarball, cached).
- **Existing docs**: README.md, CONTRIBUTING.md, SECURITY.md,
  docs/USAGE.md, docs/INSTALL.md all get edited as needed to pass —
  content changes, not just config, driven by whichever Google
  sub-rules end up kept on.
- **No dependency additions to `package.json`** beyond what each tool's
  own pinned-version invocation needs (`bunx`/pinned Docker images,
  matching how the rest of this repo's tooling is already invoked).
