# Local git hooks

This repo ships optional git hooks for contributors to install locally. They
are not required, but give you fast, local feedback before you commit,
instead of waiting on CI.

## What's here

- `pre-commit` -- blocks a commit that would introduce a restricted term
  (see below) into any staged file.
- `commit-msg` -- blocks a commit whose message contains a restricted term.

Both mirror the `RESTRICTED_TERMS` check that also runs in CI
(`.github/workflows/restricted-terms-check.yml`). The hooks are a
convenience, not the enforcement point -- they can be skipped with
`git commit --no-verify`, and only run on machines that have set them up.
CI is the actual, unbypassable gate.

## One-time setup

1. Tell Git to look here for hooks instead of the default (untracked)
   `.git/hooks`:

   ```
   git config core.hooksPath .githooks
   ```

2. Supply the restricted-term pattern list locally. The list itself is
   never committed to this repo (same reasoning as the CI secret --
   nothing here should reveal what's being checked for). Choose one:

   - Set the `RESTRICTED_TERMS` environment variable in your shell profile, or
   - Create a `restricted-terms.txt` file inside this repo's shared `.git`
     directory -- find the right path with `git rev-parse --git-common-dir`
     (this resolves correctly whether you're in the main checkout or a
     worktree, and one file there is shared by every worktree of this repo).
     One case-insensitive extended-regex pattern per line; blank lines and
     lines starting with `#` are ignored.

   If neither is set, the hooks pass trivially -- you won't be blocked from
   your first commit before you've set this up, but you also won't get any
   local protection until you do.

## Getting the pattern list

Ask a maintainer for the current list (sourced from the org's internal
term-list repo) if you don't already have it.
