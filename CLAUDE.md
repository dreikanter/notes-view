# CLAUDE.md

## Versioning

Version is set at build time via git tags and `-ldflags`. The `Version` var in
`cmd/nview/main.go` defaults to `"dev"` and is overridden by `make install` /
`make build` using `git describe --tags`.

Releases are created by dedicated release PRs. Regular PRs do not bump the
version number. On release PR merge, GitHub Actions (`.github/workflows/tag.yml`)
reads the topmost numeric `## [X.Y.Z]` heading from `CHANGELOG.md` and pushes
`vX.Y.Z` as a git tag when that heading changed.

## Changelog

Use `CHANGELOG.md` as the source of truth for release notes.

Rules:
- Keep an `## [Unreleased]` section at the top.
- PRs with user-visible changes should add an entry under `Unreleased`.
- Internal-only PRs may skip the changelog.
- Do not create a new version heading in regular PRs.
- A release PR converts `Unreleased` into `## [X.Y.Z] - YYYY-MM-DD` and adds a
  fresh empty `## [Unreleased]` section above it.
- One release may include multiple PRs.
- Reference PR numbers (`[#N]`) in changelog bullets when known and add links at
  the bottom.
- It is acceptable to open a PR with an `Unreleased` entry that does not yet
  include its PR number; add the PR reference in a follow-up commit after GitHub
  assigns the number when the entry should reference the PR.

Version bump guidance:
- Patch: fixes, docs, small behavior improvements, and internal changes worth
  releasing.
- Minor: new commands, flags, public APIs, or meaningful new behavior.
- Major: breaking CLI, config, schema, or Go API changes.

## Pull Requests

- Keep PR descriptions lean. Summarize the change in a few bullets; do not pad with implementation details that the diff already shows.
- Reference all related issues, PRs, and other resources when any exist. Use the `References` section with the appropriate relationship (`closes`, `relates to`, `depends on`, `blocked by`).
- Remove the `References` section entirely when there are no references — do not leave it empty.
