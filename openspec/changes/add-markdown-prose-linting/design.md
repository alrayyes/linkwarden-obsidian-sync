## Context

`scaffold-go-cli` (git.higherlearning.eu, cloned and read directly while
designing this) already runs this exact four-tool pipeline, battle-tested
across several repos including this project's own AUR/Nix work's
maintainer's other CLIs. Adapting it beats building from scratch — see
proposal.md for why the pipeline itself is needed.

Confirmed live against the scaffold's actual files, not assumed:

- Plain `vale <files>` (no `--minAlertLevel` flag) exits 0 on a
  warning-only finding and non-zero once an error-level finding is
  present — `MinAlertLevel = warning` in `.vale.ini` controls what's
  *reported*, not the exit-code threshold. This is why the scaffold's
  `pre-commit` hook can run bare `vale {staged_files}` while `pre-push`
  and CI use a more careful two-pass script (report everything, then a
  second `--minAlertLevel=error` pass decides pass/fail) — the second
  pass exists for a clean report artifact and an explicit gate, not
  because plain `vale`'s exit code is unreliable.
- `scripts/lint-mechanics.sh`'s `UPPERCASE_SENTENCE_START` in
  `disabledRules` independently confirms this repo's own spike finding
  (`## go install` tripping that exact rule) — the scaffold hit the same
  thing and already turned it off.

This repo is GitHub-primary (`github.com/alrayyes`), not Forgejo, so
`.forgejo/workflows/prose.yml` needs translating to GitHub Actions
syntax, not copying — same job split (`mechanics`, `style`), same
scripts, different `uses:` refs and pin format (this repo already has
its own pinned `actions/checkout`/`actions/setup-go` SHAs to reuse).

## Goals / Non-Goals

**Goals:**
- Adapt the scaffold's config files, scripts, and CI/hook wiring to this
  repo, translating Forgejo-specific pieces to what `ci.yml`/`lefthook.yml`
  already use here.
- Build this repo's own House vocabulary from scratch — the scaffold's
  `accept.txt` covers its own tooling (goreleaser, golangci-lint,
  semantic-release), not this repo's (AUR, nfpm, nixpkgs, Linkwarden,
  Obsidian, systemd, PKGBUILD...).
- Get README.md, CONTRIBUTING.md, SECURITY.md, docs/USAGE.md, and
  docs/INSTALL.md passing clean.

**Non-Goals:**
- Changing this repo's Go-tooling hooks (gofmt, golangci-lint, go test) —
  those already run through Docker per `rules/go.md`'s toolchain-drift
  reasoning, a deliberate divergence from the scaffold's host-native
  approach that predates this change and is out of scope for it.
- Renovate-style automated version bumps for LTeX/Vale — this repo runs
  Dependabot (GitHub-primary), which can't do the scaffold's custom
  `# renovate:` regex-manager comments. Matches this repo's existing
  precedent (`govulncheck@v1.7.0` in `ci.yml` is already a manually
  maintained pin, not Dependabot-watched) rather than a gap unique to
  this change.
- A `styles/config/vocabularies/House/reject.txt` — the scaffold's is
  empty with a comment explaining why (styles already cover the usual
  suspects); start the same way and add entries only if a real case
  shows up.

## Decisions

**British English (en-GB), matching the scaffold.** This repo's own docs
don't have enough spelling-sensitive words to independently confirm a
convention — a spike found exactly one hit ("license," American) across
all five docs. Asked directly rather than guessed from that thin signal:
British, for consistency with the scaffold and this maintainer's other
repos. Sets LTeX's `ltex.language` to `en-GB` and keeps `Google.Quotes`
and `OXFORD_SPELLING_Z_NOT_S` disabled the same way the scaffold does.

**Adopt the scaffold's Google sub-rule decisions in full, not just
`Google.EmDash`.** The spike that found `Google.EmDash` fighting this
repo's spaced-em-dash voice only checked that one rule; the scaffold has
already made five more of these calls for the same underlying reason
(CLAUDE.md's shared "Writing for a reader" voice, common to every repo
this maintainer writes prose linting for):
`Google.We`, `Google.FirstPerson`, `Google.Contractions` off (second-person,
contraction-heavy voice, not narrated as "we"); `Google.Headings` off
(this repo's own headings are a mix of sentence-style and stable
title-case section names like "Nix / NixOS" — neither reads as the
"title case is wrong" case Google's rule targets); `Google.EmDash` off
(confirmed independently). Reusing the scaffold's own written rationale
in each config comment rather than re-deriving it.

**Separate `prose.yml` workflow, not folded into `ci.yml`.** Matches the
scaffold: a `paths:` filter (`**/*.md`, `.vale.ini`, `.ltex.json`,
`styles/**`, the workflow file itself) means it only runs when something
prose-relevant actually changed, not on every Go-only push — this repo's
`ci.yml` already has no such filter and runs unconditionally, so adding
Vale/LTeX there would pay their cost (LTeX's cold-cache case especially)
on commits that never touch a doc.

**`scripts/lint-mechanics.sh` and `scripts/lint-prose.sh`, not inlined
commands.** One script each, called identically by the hook and the CI
job, so the two can't check a different file list or disagree about
what fails — the scaffold's own header comments explain this is exactly
the drift this arrangement exists to prevent. Copied near-verbatim; the
only repo-specific edit is the LTeX version pin comment (no `# renovate:`
annotation, since Dependabot won't act on it — see Non-Goals) and the
Docker image tag in the pre-commit `docker-build` job name
(`scaffold-go-cli` → `linkwarden-obsidian-sync`, already how this repo's
existing hadolint/docker-build jobs name their tags).

**`markdownlint-cli2` and `prettier` become real devDependencies.** This
repo's `package.json` currently only carries commitlint and lefthook
(it exists solely so those resolve, per its own description). Both new
tools get added as pinned exact versions, matching the scaffold's own
pins as a starting point, then whatever's actually current when this
lands (`bun add -D` decides the pin, per `dependencies.md`, not this
document).

## Risks / Trade-offs

- **Vocabulary work is the real unknown-sized part of this change.**
  The spike found 7 entries from LTeX and 25+ from Vale, across only
  five docs — actually building `accept.txt` and `.ltex.json`'s
  dictionary to a clean run is iterative (run, read the finding, decide
  vocabulary vs. genuine fix, repeat) and can't be fully sized until
  it's actually done. Mitigation: tasks.md breaks this out as its own
  step per tool rather than one "add vocabulary" line, so the iteration
  is visible as it happens rather than hidden inside a single task.
- **LTeX's ~300MB download makes the first `pre-push` after this lands
  slow for anyone who hasn't run it before.** Matches the scaffold's own
  accepted trade-off (cached outside the repo after that, ~10s
  thereafter) — not something this change can avoid, only document
  clearly in CONTRIBUTING.md's "Getting set up" section.
- **The scaffold itself has a gap this change shouldn't quietly
  replicate**: `lint-mechanics.sh`/`lint-prose.sh`'s `git ls-files '*.md'
  | grep -v ...` has no guard against an empty result, which
  `rules/tooling.md`'s own "git inside a container-as-root job" section
  calls out as a real, recurring failure mode (a checker's own usage
  error silently misread as "found problems"). Neither script actually
  runs inside a container-as-root job here, so the specific failure mode
  that rule names doesn't apply verbatim — but the general principle
  does. Worth a one-line guard when adapting (`[ -n "$files" ] || { echo
  "no markdown files found" >&2; exit 1; }`) rather than copied over
  silently; flagged back to the scaffold separately (`sync-scaffolds`
  territory, not this change's job to fix upstream).
