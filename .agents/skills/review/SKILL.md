---
name: review
description: Code review of a pull request, branch, or diff. Covers technical, product, and risk perspectives in one pass and produces a consolidated report. Use when asked to review a PR, branch, or code changes.
---

# Code Review

Evidence-based and blunt. Every finding references a specific `file:line`, function, or component. NEVER sugarcoat, NEVER pad with praise, NEVER report a concern that is not grounded in the diff or the codebase. Style preferences are not defects — but a violation of `AGENTS.md` or `CODESTYLE.md` is a convention finding, not a preference.

## If you are the diff's author

If you wrote the diff you are reviewing now, say so explicitly in the report's Verdict and treat this pass as necessary but insufficient. `agent-code-review/SKILL.md` covers why self-review inherits your own design assumptions and what to recommend instead — read it, don't re-derive it here.

## Before reviewing

1. Ask the user for numbered acceptance criteria (DoD). If there are none, derive them from the PR description or the linked issue, mark them `(inferred)`, and proceed — do not stall, and do not invent criteria silently.
2. Resolve the base first: `git fetch`, then diff against the branch the PR actually targets. PRs target `main`; `release-v*` branches are owned by release-please, not a review base. State the resolved base in the report. For uncommitted work, review `git diff` / `git diff --cached` instead.
3. Read every changed file. Then trace callers of the changed exported symbols whose signature or behavior changed, and of anything crossing a persistence boundary — via LSP call hierarchy and references, not grep.
4. For 10+ changed files, split the reading by area (e.g. new files, storage/cleanup, build pipeline) across subagents if your harness has them, and synthesize the findings yourself.
5. If the worktree holds the branch and `task` works, run the relevant module build and tests (`task server:build` / `task server:test:unit`, `task client:build` / `task client:test:unit`, `task release:build`, `task e2e:test:e2e`), never raw `go` — a review that never compiled the change is an opinion. The server Vault plugin is Linux-only and links CGO (delivery-kit-sdk ELF signing), so it will not compile on macOS. NEVER run `task format`: it would rewrite the diff under review.

## Technical perspective

Code structure and correctness only — user impact belongs to the product perspective.

- Conventions: `AGENTS.md` and `CODESTYLE.md` are the standard. This project prefers a bit of duplication over abstraction and minimizes interfaces and generics — flag deviations in either direction, and NEVER report duplication as a DRY defect on its own.
- Correctness: error wrapping and discarded errors, context propagation and cancellation, goroutine and errgroup ownership, nil map writes, typed-nil interfaces.
- Security: least privilege, input validation, secret handling, container security.
- Observability: when deploy or registry operations fail, is the cause visible in the logs?
- Testability: can the change be exercised without a cluster or a registry?
- Consistency with patterns already established in the project.

Cover the ones the diff actually touches; stay silent about the rest.

## Tests as evidence

Passing tests, high coverage, and the author's confidence are not evidence of correctness, whoever wrote the diff. Read `test-the-tests/SKILL.md` and run its mutation loop against every load-bearing test: mutate the implementation and confirm the test fails, rather than reading the assertions and trusting they'd catch a regression. This is not optional — skipping it because the tests "look thorough" is exactly the failure mode it exists to catch.

If the diff's author is an agent, or the diff touches tests or verification infrastructure, also read `agent-code-review/SKILL.md` — it covers check-gaming detection (weakened assertions, quietly skipped tests, mocked-out critical behavior, and more) in one place, so this list doesn't drift from it again.

## Product perspective

What the change does for the user — not how the code is written.

- User impact: CLI UX, error messages, flag names, defaults, output formatting, breaking changes.
- Completeness: edge cases (dry-run, force, conflicting flags, empty states).
- Consistency: matches existing `trdl` client and `trdl-vault` CLI conventions.
- CLI surface: a client flag needs its `TRDL_*` env counterpart (`trdl-vault` uses `TRDL_VAULT_*`); a renamed or removed flag needs a deprecation path; exit codes are a user contract, so a schema change is breaking.
- Documentation: the CLI reference under `docs/` is generated, so the fix for a stale page is `task docs:gen`, never a hand edit, and `CHANGELOG.md` is owned by release-please — a hand edit to it is itself a defect.

## Risks

Derive risks from the technical and product findings plus the diff — including compound ones, where a technical flaw produces a product gap or an operational hazard. Likelihood is Likely/Possible/Unlikely, severity is Critical/High/Medium/Low; be realistic, do not inflate. Every risk needs a concrete location.

Classify each risk as Technical, Security, UX/Product, or Operational, and report risks only when they exist — an empty matrix is noise. A `go.mod` bump of the shared `common-go` library or of the Vault/TUF dependencies carries the widest blast radius here — it silently changes release, signing, or update behavior for every user.

## Gotchas

- trdl is a multi-module project (`server/`, `client/`, `release/`, `e2e/`) — a change to a shared contract may need coordinated edits across modules.
- The server is a Vault plugin: it verifies operations through GPG signatures and maintains a TUF repository. Any change to signature verification, key rotation, or TUF metadata is a security-critical path — inspect it directly.
- Persisted formats (TUF metadata and keys, release bundles, S3/MinIO storage layout) need backward compatibility.
- `*_linux.go` / `*_others.go` pairs and the Linux-only server plugin must stay consistent; a reviewer on macOS cannot compile the CGO server side at all.
- Use `samber/lo` helpers only in `server/` — they are not available in `client/` or `release/`.
- Build and test only via `task` commands, never raw Go tools. AI-authored tests must be `*_ai_test.go`, tagged `//go:build ai_tests`, with `TestAI_`-prefixed functions.

## Output

Print the report. Do not write it into the repository unless the user asks for a file.

```markdown
# Code Review Report

**Base:** `<resolved base branch>`
**Diff:** [X files, +Y/-Z lines]

## Verdict

- Technical: [up to 3 sentences, or `no findings`]
- Product: [up to 3 sentences, or `no findings`]
- Risk: [up to 3 sentences, or `no findings`]

## DoD Criteria

| Criteria | Inferred? | Met? | Evidence |
| :--- | :--- | :--- | :--- |
| [criterion] | yes/no | ✅/⚠️/❌ | file:line |

## Issues

- **Critical** — blocking, with file:line
- **Major** — significant concern
- **Minor** — suggestion

## Risks

Sorted by severity, then by likelihood.

| № | Risk | Type | Likelihood | Severity | Location | Circumstances | Consequences | Recommendation |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |

## Not verified

- What was not built, run, or reachable — and why (the server Vault plugin does not compile on macOS, e2e needs Linux with Docker and a running Vault/MinIO).
```

- **Recommendation** — the concrete action, with file:line references.

## Language

Headers in English, everything else in the user's language.
