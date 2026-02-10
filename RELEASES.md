# Release Process

This document describes how kvx releases are created.

## Versioning

- kvx follows Semantic Versioning (SemVer): MAJOR.MINOR.PATCH.
- Tags are prefixed with `v` (example: `v0.1.8`).

## Creating a Release

1. Ensure `CHANGELOG.md` is updated for the release.
2. Create and push a git tag:

```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

3. The GitHub Actions release workflow will run GoReleaser to build and publish artifacts.

## Release Artifacts

- Archives for linux, darwin, and windows
- SHA256 checksums

## Notes

- GoReleaser config is in `.goreleaser.yaml`.
- CI runs tests before publishing.
