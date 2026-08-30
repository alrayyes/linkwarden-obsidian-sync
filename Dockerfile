# syntax=docker/dockerfile:1
#
# goreleaser has already cross-compiled the binary this stage copies in —
# per this org's go-releases.md, this Dockerfile's only job is COPY, never
# `go build`.
#
# distroless static, not alpine: this tool has no runtime dependency
# beyond the binary itself now that it no longer shells out to git — the
# usual pick for a static Go binary applies. :nonroot already runs as a
# non-root user (uid 65532), so nothing to add here for that either.
FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab

COPY linkwarden-obsidian-sync /usr/local/bin/linkwarden-obsidian-sync

ENTRYPOINT ["/usr/local/bin/linkwarden-obsidian-sync"]
