## 1. Config files

- [x] 1.1 Add `.prettierrc.json` (`proseWrap: preserve`, `tabWidth: 2` for
      `*.md`) and `.prettierignore` (node_modules, dist, bun.lock, go.sum,
      CHANGELOG.md, `styles/Google/`, `styles/proselint/`), adapted from
      scaffold-go-cli's own — verify `bunx prettier --check
      "**/*.{md,yml,yaml}"` runs without error (findings expected until
      task 6)
- [x] 1.2 Add `.markdownlint-cli2.yaml` extending
      `markdownlint/style/prettier`, `line-length` re-enabled under its
      alias (`line_length: 80`, `tables: false`, `code_blocks: false`),
      globs scoped to `**/*.md` minus `CHANGELOG.md` and Vale's
      downloaded style directories — verify `bunx markdownlint-cli2
      "**/*.md"` runs without a config error
- [x] 1.3 Add `.vale.ini` — `StylesPath = styles`, `MinAlertLevel =
      warning`, `Packages = Google, proselint`, `Vocab = House` (core
      option, above the format section), `[*.md]` block with
      `Vale.Terms = YES` and `Google.We`/`Google.FirstPerson`/
      `Google.Contractions`/`Google.Headings`/`Google.EmDash`/
      `Google.Quotes` all off — verify `vale sync && vale --version`
      succeeds
- [x] 1.4 Create `styles/config/vocabularies/House/accept.txt` (start
      empty beyond a header comment — task 6 fills it in from real
      findings, not guessed in advance) and an empty
      `styles/config/vocabularies/House/reject.txt` with the scaffold's
      own explanatory comment
- [x] 1.5 Add `.ltex.json` — `ltex.language: en-GB`,
      `ltex.disabledRules.en-GB` seeded with the scaffold's 13-rule list
      (`PASSIVE_VOICE`, `UPPERCASE_SENTENCE_START`, `DASH_RULE`,
      `EN_QUOTES`, `OXFORD_SPELLING_Z_NOT_S`, and the rest), empty
      `ltex.dictionary.en-GB` array (task 6 fills it in) — verify the
      file is valid JSON
- [x] 1.6 Add `.gitignore` entries for `styles/Google/` and
      `styles/proselint/` (Vale's downloaded packages, not committed —
      matches `.prettierignore`'s same exclusion)

## 2. Scripts

- [x] 2.1 Add `scripts/lint-mechanics.sh`, adapted from scaffold-go-cli:
      pinned `ltex-ls-plus` version (comment notes no `# renovate:`
      annotation — this repo runs Dependabot, which won't act on it),
      cache under `$XDG_CACHE_HOME`, bundled-JDK `JAVA_HOME` resolved by
      glob, `git ls-files '*.md' | grep -v '^CHANGELOG.md$'` for the file
      list with an explicit empty-list guard (design.md's flagged gap —
      `[ -n "$files" ] || { echo "no markdown files found" >&2; exit 1;
      }`), non-zero exit check (not `-eq 3`) — verify it runs clean
      against a single throwaway file with no findings
- [x] 2.2 Add `scripts/lint-prose.sh`, adapted the same way: `vale sync`
      if the style dirs are missing, the same file-list-with-guard
      pattern as 2.1, `vale --output=line` piped to `${VALE_REPORT:-
      /dev/null}`, a second `vale --minAlertLevel=error` pass as the
      actual gate — verify it runs clean against a single throwaway file
      with no findings
- [x] 2.3 `chmod +x` both scripts and verify they're executable from a
      fresh clone (`git diff --stat` shows the mode bit, or `ls -l`)

## 3. Dependencies and hook wiring

- [x] 3.1 Add `markdownlint-cli2` and `prettier` as pinned exact-version
      `devDependencies` in `package.json`, add `format:check`, `lint:md`,
      `lint:prose`, `lint:mechanics` scripts — verify `bun install`
      succeeds and each script runs (failures from real findings are
      expected until task 6)
- [x] 3.2 Add `pre-commit` jobs to `lefthook.yml`: `prettier` (fixer,
      `glob: "*.{md,yml,yaml}"`, `stage_fixed: true`), `markdownlint`,
      `vale` (bare `vale {staged_files}`, no `--minAlertLevel` flag —
      design.md confirmed this already exits 0 on warning-only findings)
      — verify `lefthook run pre-commit` picks up all three job names
- [x] 3.3 Add `pre-push` jobs: `prettier` (`bun run format:check`),
      `markdownlint` (`bun run lint:md`), `vale` (`bun run lint:prose`,
      whole-tree not staged-only), `mechanics` (`bun run lint:mechanics`)
      — verify `lefthook run pre-push` picks up all four job names

## 4. CI

- [x] 4.1 Add `.github/workflows/prose.yml`: translated from
      scaffold-go-cli's `.forgejo/workflows/prose.yml` — `paths:` filter
      on `**/*.md`, `.vale.ini`, `.ltex.json`, `styles/**`, the workflow
      file itself; `mechanics` job running `./scripts/lint-mechanics.sh`
      (no separate Java setup — the script's own bundled-JDK resolution
      covers it); `style` job installing Vale via `go install
      github.com/errata-ai/vale/v3/cmd/vale@<pinned>` (this repo already
      has Go set up in CI) then running `./scripts/lint-prose.sh` with
      `VALE_REPORT` piped into `$GITHUB_STEP_SUMMARY` — use this repo's
      own already-pinned `actions/checkout` SHA, not the Forgejo one —
      verify with `actionlint` and a plain YAML parse
- [x] 4.2 Add `prose / mechanics` and `prose / style` to branch
      protection's required status checks (same API call pattern already
      used for `packages`/`aur-package`/`nix-flake`) — verify via `gh api
      repos/alrayyes/linkwarden-obsidian-sync/branches/main/protection/required_status_checks`
      after the first real run reports both contexts

## 5. Get the real docs passing clean

- [x] 5.1 Run `bun run lint:md` against README.md, CONTRIBUTING.md,
      SECURITY.md, docs/USAGE.md, docs/INSTALL.md; fix genuine structure
      issues found (not suppress) — verify exit 0
- [x] 5.2 Run `bun run format:check`; if it fails, run the fixer
      (`bunx prettier --write`) and commit the reformatted files as their
      own change — verify `format:check` then exits 0
- [x] 5.3 Run `bun run lint:mechanics`; for each finding, decide per
      design.md's risk note — real mistake (fix the prose) vs. this
      repo's own vocabulary (add to `.ltex.json`'s `ltex.dictionary.en-GB`)
      — iterate until exit 0. Expect at minimum: Linkwarden, Codecov,
      systemd, toolchain, PKGBUILD, nixpkgs, frontmatter (already found
      by the earlier spike)
- [x] 5.4 Run `bun run lint:prose`; for each `Vale.Spelling` finding, add
      to `styles/config/vocabularies/House/accept.txt` (correct casing,
      since `Vale.Terms = YES` holds the doc to exactly that casing
      everywhere); for each remaining Google/proselint finding at error
      level, fix the prose or decide it's a rule to turn off in
      `.vale.ini` (with a comment explaining why, matching the existing
      six) — iterate until `vale --minAlertLevel=error` exits 0. Expect
      at minimum the 25+ terms the earlier spike found (hadolint,
      Dockerfile, commitlint, lefthook, Postgres, Meilisearch, resync,
      config, env, repo, enum, CVEs, ci, and more)
- [x] 5.5 Run the full local hook set end to end (`lefthook run
      pre-commit`, `lefthook run pre-push`) against a clean checkout —
      verify both exit 0 with no further fixes needed

## 6. Documentation

- [x] 6.1 Add a "Prose and docs linting" subsection to CONTRIBUTING.md's
      "Getting set up" section — the four tools, the ~300MB LTeX
      download's one-time cost, where the House vocabulary lives and how
      to extend it — verify by reading it as a new contributor would:
      does it say what to run and what to expect?

## 7. Final verification

- [x] 7.1 Open the pull request, confirm all CI jobs — including the two
      new `prose.yml` jobs — pass on the real GitHub Actions runner, not
      just locally
- [x] 7.2 Confirm issue #45's acceptance criteria hold against the
      merged state: a `line-length` violation fails structure; "an
      university"/"the the" fails mechanics; a passive-voice sentence
      warns without failing; `PASSIVE_VOICE` is off in LTeX; CI shows
      `mechanics` and `style` as separately named checks.

      This check found a real gap after #70 had already merged (auto-merge
      fired the moment CI went green, before this check ran): neither
      Google nor proselint actually has a passive-voice rule — a
      deliberately passive test sentence got zero Vale findings. Fixed in
      a follow-up PR adding the `write-good` package (confirmed live:
      `write-good.Passive` is what LTeX's own disabled `PASSIVE_VOICE`
      rule was actually standing in for) — see PR history on `main` for
      both.
