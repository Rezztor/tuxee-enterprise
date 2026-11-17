Parent DOX: [project](../AGENTS.md).

# Purpose

The release workflow, release-note generation, repository sponsorship and presentation assets.

# Ownership

Own `workflows/release.yml`, release-note generation and `FUNDING.yml`. Root owns Dockerfile and installer; internal/update owns the
release-asset contract. `resources/` owns the optimized README video hero, screenshots, infographics and their
thumbnails.

# Local Contracts

- Rezztor build: one workflow, `release.yml`, on GitHub's `ubuntu-22.04` runner. A pull request builds and tests;
  a pushed `vMAJOR.MINOR.PATCH` tag also builds `tuxee-linux-amd64` with the tag in `main.version` and publishes it
  with `checksums.txt` as a GitHub release for `install.sh`. Upstream's Docker image builds and nightly rebuild were
  removed: Rezztor ships for customers' own Linux hosts, not containers. The binary links the runner's glibc (2.35),
  so customers need glibc 2.35 or newer.
- `FUNDING.yml` uses Agent Zero's sponsorship configuration (`github: agent0ai`).
- Release notes include every commit subject/body since the previous reachable stable version tag, or all commits
  for the first tag.

# Work Guidance

No development-branch publication or external release-note generator.

# Verification

`python3 .github/release-notes.py --self-test`.

# Child DOX Index

- [resources/AGENTS.md](resources/AGENTS.md): Optimized video hero, screenshots, infographics and thumbnails linked from the README.
