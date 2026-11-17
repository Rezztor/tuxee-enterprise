Parent DOX: [project](../AGENTS.md).

# Purpose

Release-note generation, repository sponsorship and presentation assets.

# Ownership

Own release-note generation and `FUNDING.yml`. Root owns Dockerfile and installer; internal/update owns the
release-asset contract. `resources/` owns the optimized README video hero, screenshots, infographics and their
thumbnails.

# Local Contracts

- Rezztor build: no Actions workflows. Upstream's tag release and nightly Docker rebuild were removed because Rezztor
  ships for customers' own Linux hosts, not containers, and publishes nothing to Docker Hub.
- `FUNDING.yml` uses Agent Zero's sponsorship configuration (`github: agent0ai`).
- Release notes include every commit subject/body since the previous reachable stable version tag, or all commits
  for the first tag.

# Work Guidance

No development-branch publication or external release-note generator.

# Verification

`python3 .github/release-notes.py --self-test`.

# Child DOX Index

- [resources/AGENTS.md](resources/AGENTS.md): Optimized video hero, screenshots, infographics and thumbnails linked from the README.
