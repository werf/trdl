---
name: git-conventions
description: Git workflow conventions for commits, branches, and pull requests. Load this skill when creating commits, branches, or PRs.
---

## Git Conventions

Load this skill when creating commits, naming branches, or opening pull requests.

### Commit Message

```
<type>(<scope>): <subject>
<BLANK LINE>
<body>
```

#### Type

| Type | Purpose |
|------|---------|
| **feat** | New features or capabilities that enhance the user's experience |
| **fix** | Bug fixes that enhance the user's experience |
| **refactor** | Code changes that neither fix a bug nor add a feature |
| **docs** | Updates or improvements to documentation |
| **test** | Additions or corrections to tests |
| **chore** | Updates that don't fit into other types |

#### Scope

Scope identifies the area of the project affected by the change. It is context-dependent and derived from the project structure. Common top-level scopes for maintaining code quality and development workflow:

- **ci** — CI/CD pipeline
- **release** — release process
- **dev** — development workflow
- **deps** — dependencies

#### Subject

- Imperative, present tense: "change" not "changed" nor "changes"
- Don't capitalize the first letter
- No dot (.) at the end

#### Body

- Imperative, present tense (same as subject)
- Include the motivation for the change and contrast with previous behavior

### Branch Name

```
<type>/<scope>/<short-description>
```

- Use only the **top-level scope** (no nested/multiple scopes)
- `<short-description>` is kebab-case, concise, hyphen-separated

Examples:
```
feat/api/add-user-endpoint
fix/auth/token-refresh
refactor/core/simplify-parser
```

### Pull Requests

See the `pull-request` skill for PR title and description conventions (draft by default, `<type>(<scope>): <subject>` title mirroring the main commit, mandatory Summary / Key changes / Why / Review focus structure).

### Before pushing

- ALWAYS check the current branch (`git branch --show-current`) before pushing — don't assume it matches what you intend to push. A repo can have a stale local `main` far behind `origin/main`, or you can be checked out on someone else's WIP topic branch without noticing.
- NEVER push directly to a shared/team repo's default branch (one you don't solely own — e.g. an open-source project, a company repo with other contributors). Branch from the current `origin/<default>` and open a PR instead, even for a small or "obviously safe" change.
- If a push is rejected as non-fast-forward, don't force it — fetch and figure out why the ref diverged (stale local branch, someone else's history, wrong ref name) before deciding how to proceed.
