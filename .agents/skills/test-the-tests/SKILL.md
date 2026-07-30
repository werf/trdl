---
name: test-the-tests
description: >
  Verify that a test actually falsifies the behavior it claims to cover, via real mutation
  (not just reading the assertions). Use right after writing or changing tests — before
  considering that work done — when auditing an existing suite's real coverage, as a step
  inside a broader code review, or when invoked as /test-the-tests. Passing and high
  coverage are not evidence; a test that survives a plausible fault is weak regardless of
  whether it currently passes.
---

# Test the Tests

A test's value is what it would catch, not what it currently asserts. Don't stop at "does
this read as correct" — ask:

> What fault would this test fail to catch?

This applies to your own tests as much as anyone else's. Writing a test and watching it
pass proves the test *runs*; it does not prove the test *discriminates* correct from
incorrect behavior. Treat "I just wrote this test" as no more trustworthy than "someone
else wrote this test."

## The mutation loop

For each test that carries real weight (guards a fix, a regression, an invariant — not a
trivial getter):

1. Pick the smallest plausible fault: invert a condition, change `<` to `<=`, remove a
   validation, suppress an error, skip a side effect, hardcode/return a constant, revert to
   the prior (buggy) implementation.
2. Apply it directly to the implementation.
3. Run the test. It must fail, with a message that points at the actual broken behavior —
   not a crash unrelated to the property under test.
4. Revert the fault immediately, confirm the tree is clean (`git diff`/`git status`), and
   confirm the suite passes again.

If running the mutation isn't practical (expensive integration setup, no easy revert),
name the smallest mutation that should be tried instead of skipping the exercise — that
name is itself the finding when no one can say what it is.

Do not leave mutated code in the repository, even temporarily between steps — always
restore from a real diff/backup and re-verify a clean tree before moving on.

## Common ways a test looks strong but isn't

- **The assertion is satisfied by more than the fix.** A chain assertion like
  `index(a) < index(b) < index(c)` can hold under both the correct implementation and the
  bug it's meant to catch, if the specific scenario doesn't force them to disagree. Reshape
  the scenario (add a case whose correct answer differs from what the bug would produce)
  rather than trusting that "the numbers came out right" once.
- **It re-asserts a second recording of the same event instead of the event itself.**
  Comparing two independently-recorded orderings (e.g. two separate mutex-protected
  append lists) is often racier and weaker than asserting a property that's true by
  construction (a monotonic counter, a guaranteed dependency order).
- **It only exercises the unit in isolation**, never the real wiring that a regression
  would actually break (e.g. calling a scheduler's internal methods directly instead of
  going through the real caller that's supposed to invoke them). Prefer driving the actual
  entry point end-to-end when the risk is in the wiring, not just the unit.
- **It asserts implementation trivia** (a mock was called N times, an internal helper ran)
  instead of externally observable behavior a user or caller would notice.
- **Coverage number, not falsifiability.** A line being executed says nothing about whether
  a wrong value on that line would be caught.

## Output

For each test you verified this way, state: the fault you tried, whether it failed as
expected, and if not, the smallest concrete change to the test that would make it
discriminate. For tests you didn't (couldn't) mutate, name the smallest fault that should
be tried next, rather than asserting confidence without it.
