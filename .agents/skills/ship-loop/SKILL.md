---
name: ship-loop
description: Take one software task from request to a reviewed, merge-ready draft PR without stopping halfway. Product goal first, a plan with a "not building" list, plan review, the smallest root-cause implementation, tests, real browser verification with screenshots (up to 10 fix rounds), a readiness verdict, 3 independent local reviewers on 2+ models, one draft PR with a human-readable body, and a CI/review fix loop. Use for "implement and ship", "take this to a PR", "fix this PR until green".
---
# Ship Loop

## Ground rules
- Stop only for human gates: merge, production, new dependencies, public APIs/schemas, secrets, user data.
- If the same approach fails twice, change approach. If that fails too, stop and report.
- Nothing reaches GitHub before local review passes.

## The shiplog
`.scratch/ship/<task>.md`: goal, plan, current step, verification results, review findings and fixes, PR link. Update after every step.

## 0. Resume
If a shiplog exists for this task, continue from its current step.

## 1. Intake
Write the product goal in one user sentence and how we will see it working. Sort open questions: ones that block knowing what to build (auth, data exposure, irreversible effects) go to the human now, all at once; ones that do not (copy, naming, reversible defaults) you decide and record.

## 2. Preflight
Clean branch from the default branch, app starts with the one command, tests pass before you change anything (record failures that already exist).

## 3. Plan
- Outcome: who hits it, what triggers it, what they see now, what they should see.
- Evidence: code inspected; observed vs guessed.
- Changed contract: inputs, outputs, states crossing the boundary.
- Bad-day cases: reload, retry, duplicate, cancel, partial failure.
- Proof: the 1-3 user jobs you will prove, and how.
- NOT building: what this change deliberately leaves out.

## 4. Plan review
One fresh-context reviewer checks the plan against the goal. Fix before coding.

## 5. Implement
Smallest change at the root cause. Reuse existing patterns. Add or update tests that fail without the change.

## 6. Diff hygiene
Read your own diff: no debug code, no unrelated edits, no secrets, no commented-out code.

## 7. Verify
Run lint, typecheck, tests. All green.

## 8. Real product check (up to 10 rounds)
Use the verify skill. Start the app, do the user job in a real browser, take before/after screenshots into `.scratch/ship/<task>/`. Open every screenshot and judge it: is the requested behavior visible and correct? If anything is off, fix and retest everything.

## 9. Readiness gate
Write a verdict: ready or not, with the evidence. Not ready means back to step 5.

## 10. Local review gate
Rebase on the latest default branch, commit, and run everything CI runs locally first. Run the code-review skill with 3 independent reviewers in fresh sessions, on 2+ models when available. Fix, then re-review until all 3 approve. Max 10 rounds for normal findings; security findings are never capped. If independent reviewers cannot run, open the PR as a draft marked "review unavailable"; never count self-review as approval.

## 11. Publish once, as a draft
Push the branch, open ONE draft PR with a body from the pr-body skill and the screenshots.

## 12. CI and review loop
If GitHub finds something, the local gate missed it: reproduce locally, fix, rerun the gate, note why it was missed. Fix every CI failure and review comment at the root cause, re-verify, push. Mark ready when CI is green.

## 13. Exit
Report `ready_to_merge` (a real PR URL, current evidence proves the goal), or `blocked` with the exact reason and what the human must decide.
