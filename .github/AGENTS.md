Parent DOX: [project](../AGENTS.md).

# Purpose

The release workflow, release-note generation, repository sponsorship and presentation assets.

# Ownership

Own `workflows/release.yml`, release-note generation and `FUNDING.yml`. Root owns Dockerfile and installer; internal/update owns the
release-asset contract. `resources/` owns the optimized README video hero, screenshots, infographics and their
thumbnails.

# Local Contracts

- Rezztor build: one workflow, `release.yml`. A pushed `vMAJOR.MINOR.PATCH` tag runs on Rezztor's self-hosted
  x64 runner, tests, builds `tuxee-linux-amd64` with the tag in `main.version`, and publishes it with `checksums.txt`
  as a GitHub release for `install.sh`. Upstream's Docker image builds and nightly rebuild were removed: Rezztor ships
  for customers' own Linux hosts, not containers. The runner needs build-essential, libpam0g-dev, musl-tools and gh;
  the binary links the runner's glibc, so customers need that glibc or newer.
- `FUNDING.yml` uses Agent Zero's sponsorship configuration (`github: agent0ai`).
- Release notes include every commit subject/body since the previous reachable stable version tag, or all commits
  for the first tag.

# Work Guidance

No development-branch publication or external release-note generator.

# Verification

`python3 .github/release-notes.py --self-test`.

# Child DOX Index

- [resources/AGENTS.md](resources/AGENTS.md): Optimized video hero, screenshots, infographics and thumbnails linked from the README.
