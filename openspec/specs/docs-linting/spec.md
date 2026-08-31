# docs-linting Specification

## Purpose

Catches broken structure, grammar and spelling mistakes, and off-voice
prose in this repo's Markdown documentation before it merges, the same
way code is linted, rather than leaving it to whoever happens to notice.

## Requirements

### Requirement: Structure is checked
Every tracked Markdown file SHALL pass markdownlint-cli2's structure
checks — including a re-enabled `line-length` rule — in both local hooks
and CI.

#### Scenario: A line exceeds the configured length
- **WHEN** a Markdown file contains a line longer than the configured
  `line-length` limit
- **THEN** markdownlint-cli2 fails and reports the `line-length` rule
  specifically, not a silently-inherited pass

#### Scenario: Structure passes clean
- **WHEN** `lefthook run pre-commit` or `lefthook run pre-push` runs
  against this repo's existing tracked Markdown files
- **THEN** markdownlint-cli2 exits 0

### Requirement: Layout is normalized automatically
Prettier SHALL reformat every tracked Markdown file (and the YAML files
it also covers) to a consistent layout, fixed automatically at commit
time and only checked (never rewritten) at push time and in CI.

#### Scenario: A commit touches a Markdown file with inconsistent layout
- **WHEN** a file staged for commit has layout Prettier would reformat
- **THEN** `pre-commit` rewrites it and restages the result before the
  commit completes

#### Scenario: A push carries layout Prettier has not seen
- **WHEN** `lefthook run pre-push` or CI runs `prettier --check` against
  a file Prettier would still reformat
- **THEN** the check fails without modifying the file

### Requirement: Grammar and spelling mistakes fail the build
ltex-cli-plus SHALL check every tracked Markdown file for grammar and
spelling in `pre-push` and CI, and a finding SHALL fail the run (exit
code 3, not 0) regardless of the finding's own reported severity level.

#### Scenario: A grammar mistake is present
- **WHEN** a Markdown file contains a rule violation ltex-cli-plus's
  default rule set catches — e.g. an article/noun mismatch like "an
  university", or a repeated word like "the the"
- **THEN** ltex-cli-plus exits non-zero (3) in both `pre-push` and CI

#### Scenario: This repo's own vocabulary appears in prose
- **WHEN** a Markdown file uses a term the project's own vocabulary
  covers — a product name, a tool name, a piece of this repo's own
  jargon
- **THEN** ltex-cli-plus does not report it as a spelling mistake

### Requirement: Style is advisory, not a hook failure
Vale SHALL check every tracked Markdown file against this repo's chosen
style rules in `pre-commit`, `pre-push`, and CI, and a style finding
SHALL be reported as a warning without failing any of those runs.

#### Scenario: A passive-voice sentence is present
- **WHEN** a Markdown file contains a passive-voice construction Vale's
  configured styles flag
- **THEN** Vale reports it and the run it's part of still exits 0

#### Scenario: A style rule conflicts with this repo's established voice
- **WHEN** a Google style sub-rule contradicts a convention this repo's
  own writing voice deliberately uses — e.g. spaced em-dashes
- **THEN** that specific sub-rule is disabled in `.vale.ini` rather than
  the repo's prose being rewritten to match it, and rewriting is not
  required to pass the check

### Requirement: Style and mechanics do not double-report the same issue
Where Vale and ltex-cli-plus both have a rule for the same class of
issue, at most one of them SHALL have that rule enabled.

#### Scenario: A passive-voice sentence is present
- **WHEN** a Markdown file contains a passive-voice construction
- **THEN** at most one of Vale or ltex-cli-plus reports it — ltex-cli-plus's
  `PASSIVE_VOICE` rule is disabled because Vale's configured styles
  already cover it

### Requirement: CI reports which tier failed
CI SHALL run Vale and ltex-cli-plus as two separate jobs, not one script
invoking both.

#### Scenario: Only the style tier has a finding
- **WHEN** a pushed commit's Markdown passes ltex-cli-plus but Vale
  reports a style warning
- **THEN** the CI job named for Vale reflects that state and the job
  named for ltex-cli-plus is unaffected by it

### Requirement: This repo's own documentation passes clean
README.md, CONTRIBUTING.md, SECURITY.md, docs/USAGE.md, and
docs/INSTALL.md SHALL pass all four tools with no suppressions beyond
entries in the House vocabulary.

#### Scenario: The full local hook set runs against a clean checkout
- **WHEN** `bun install` has run and `lefthook run pre-commit` followed
  by `lefthook run pre-push` runs against this repo's existing tracked
  Markdown files
- **THEN** all four tools exit successfully with no manual fixes needed
  beyond what's already committed
