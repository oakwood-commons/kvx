# Open Source Readiness Plan

Checklist for preparing kvx for public release.

## Required Items

- [x] **SECURITY.md** - Vulnerability reporting policy ✓
- [x] **CODE_OF_CONDUCT.md** - Community standards (contact set)
- [x] **CHANGELOG.md** - Initial changelog created
- [x] **Release Workflow** - `.github/workflows/release.yml` for goreleaser automation
- [x] **GitHub Issue Templates** - Bug report and feature request templates
- [x] **Pull Request Template** - `.github/PULL_REQUEST_TEMPLATE.md`
- [x] **Initial Version Tag** - Tags exist (v0.1.0 through v0.1.8)

## Optional (Completed)

- [x] **CODEOWNERS** - Review routing in `.github/CODEOWNERS`
- [x] **GOVERNANCE.md** - Maintainer and decision process
- [x] **SUPPORT.md** - Support channels and guidance
- [x] **RELEASES.md** - Release process documentation
- [x] **Dependabot Config** - `.github/dependabot.yml`

## Already Complete

- [x] LICENSE (Apache 2.0)
- [x] README.md (comprehensive)
- [x] CONTRIBUTING.md (complete guidelines)
- [x] CI Workflow (tests, lint, build matrix)
- [x] All tests passing
- [x] Linting clean (0 issues)
- [x] .goreleaser.yaml configured
- [x] cliff.toml configured
- [x] Documentation (docs/ folder)
- [x] .gitignore properly configured

## Post-Public (Manual GitHub Settings)

- [ ] Enable Discussions
- [ ] Set branch protection rules
- [ ] Configure Dependabot (enable in repo settings if needed)

## Notes

- Module path: `github.com/oakwood-commons/kvx` - verify org exists
- 3 minor TODO comments in code (non-blocking)
