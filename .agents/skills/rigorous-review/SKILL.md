---
name: rigorous-review
description: Evidence-based code review of a PR, branch, or diff — technical, product, and risk perspectives in one pass, with mandatory test-quality (mutation) checks and an honest "not verified" account. Use when asked to review a PR, branch, or code changes.
---

# Rigorous Code Review

Evidence-based and blunt. Every finding references a specific `file:line`, function, or component. NEVER sugarcoat, NEVER pad with praise, NEVER report a concern that is not grounded in the diff or the codebase. Style preferences are not defects — but a violation of the project's own conventions (its `AGENTS.md` / `CLAUDE.md` / `CONTRIBUTING.md` / style guide) is a convention finding, not a preference.

## Before reviewing

1. Ask the user for numbered acceptance criteria (DoD). If there are none, derive them from the PR description or the linked issue, mark them `(inferred)`, and proceed — do not stall, and do not invent criteria silently.
2. Resolve the base first: `git fetch`, then diff against the branch this change actually targets — not blindly `main`/`master` (many projects maintain release branches; a backport's real base is the release branch, not the trunk). State the resolved base in the report. For uncommitted work, review `git diff` / `git diff --cached` instead.
3. Read every changed file. Then trace callers of changed exported symbols whose signature or behavior changed, and of anything crossing a persistence or API boundary — via LSP call hierarchy and references where available, not grep alone.
4. For 10+ changed files, split the reading by area across subagents if your harness has them, and synthesize the findings yourself.
5. Check the project's own docs (`AGENTS.md`, `CLAUDE.md`, `CONTRIBUTING.md`, a style guide, a project-specific review skill) for conventions, build/test commands, and known landmines before forming an opinion — a project's own accumulated gotchas beat generic defaults every time.
6. If the worktree holds the branch and it's buildable, actually build and run the relevant tests — a review that never compiled the change is an opinion, not a review. NEVER run an auto-formatter as part of this: it would rewrite the diff under review.

## Technical perspective

Code structure and correctness only — user impact belongs to the product perspective.

- Conventions: the project's own style guide/`AGENTS.md`/`CLAUDE.md` is the standard, not your default taste. Flag deviations from the project's stated preferences in either direction (e.g., if the project explicitly prefers duplication over abstraction, don't file duplication as a defect on its own, but do flag needless abstraction).
- Correctness: error handling and discarded errors, resource lifecycle and cancellation, concurrency ownership, nil/uninitialized state, boundary conditions.
- Security: least privilege, input validation, secret handling.
- Observability: when an operation fails, is the cause visible to whoever operates this system?
- Testability: can the change be exercised without external dependencies (network, cluster, hardware)?
- Consistency with the patterns already established elsewhere in this codebase.

Cover the ones the diff actually touches; stay silent about the rest.

## Tests as evidence (mandatory)

Passing tests, high coverage, and the author's confidence are not evidence of correctness, whoever wrote the diff. For every load-bearing test touched by or covering the diff, run the `test-the-tests` skill's mutation loop against it. This is not optional — skipping it because the tests "look thorough" is exactly the failure mode it exists to catch.

If the diff's author is an agent, or the diff touches tests or verification infrastructure, also load the `agent-code-review` skill — it adds check-gaming detection (weakened assertions, quietly skipped tests, mocked-out critical behavior, and more) and explains this review's limits when you are the diff's own author. Load it there rather than re-deriving the same checklist here, so the two stay in sync.

## Product perspective

What the change does for the user — not how the code is written.

- User impact: UX, error messages, naming, defaults, output formatting, breaking changes.
- Completeness: edge cases (empty states, conflicting options, boundary sizes).
- Consistency: matches existing conventions elsewhere in the product.
- Surface area: a public API/CLI/config change needs a migration or deprecation path; a change to machine-readable output (exit codes, structured output consumed by other tools) is a breaking change even if humans don't notice.
- Documentation: is it out of date after this change, and does the project generate it (don't hand-edit generated docs — fix the generator or source).

## Risks

Derive risks from the technical and product findings plus the diff — including compound ones, where a technical flaw produces a product gap or an operational hazard. Likelihood is Likely/Possible/Unlikely, severity is Critical/High/Medium/Low; be realistic, do not inflate. Every risk needs a concrete location.

Classify each risk as Technical, Security, UX/Product, or Operational, and report risks only when they exist — an empty matrix is noise.

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

- What was not built, run, or reachable — and why (platform limitation, missing infra, out-of-scope environment).
```

- **Recommendation** — the concrete action, with file:line references.

## Language

Headers in English, everything else in the user's language.
