---
name: code-review
description: Independent local code review before anything reaches GitHub. Runs 3 reviewers in fresh sessions (2+ different models when available), each given only the goal and the diff, never the author's conclusions. Each returns a verdict and ranked findings with file:line and a concrete failure scenario. Loops fix and re-review until all approve. Use before opening or updating a PR, or when the user says "review this", "check my diff", "is this ready".
---
# Code Review: three fresh eyes before GitHub

## Why
The author's session is biased toward its own plan. Fresh reviewers on different models catch what the author cannot see. Local review is cheap; CI and human review rounds are not.

## 1. Prepare the packet
- The product goal in one sentence and the NOT building list.
- The diff against the default branch (`git diff <base>...HEAD`).
- How to run the tests and the app.
Do NOT include your own opinion of the change, your verification summary or "this should be fine". Reviewers must form their own view.

## 2. Start 3 reviewers
Each in a fresh session (sub-agent), read-only, on 2+ models when available. Give each the same packet and this brief:
"Review this diff against the goal. Read the changed files fully and their callers. Run the tests. Look for, in order: correctness bugs, security (auth, injection, secrets, user data), data loss or migrations, broken edge cases (empty, error, concurrency, mobile), missing tests for the changed behavior, scope creep beyond the goal, then simplification. Every finding needs file:line, severity (critical, major, minor), and a concrete failure scenario: input or state, then the wrong result. No scenario means no finding. Do not report style taste. End with exactly: VERDICT: APPROVED or VERDICT: CHANGES REQUESTED, and FINDINGS: <count>."

## 3. Judge the findings yourself
For each finding: reproduce or trace it. Real: fix at the root cause and add a test. Wrong: write one line on why in the shiplog. Never dismiss a security finding without proof.

## 4. Re-review
Send the new diff (delta since last round plus the full diff) to the reviewers that requested changes, again with no author commentary. They also confirm earlier findings are fixed.

## 5. Exit rules
- Ready when all 3 return APPROVED in the same round on the current commit.
- Max 10 rounds for normal findings, then report what is still open.
- Security and data-loss findings are never capped: fix them or block the PR.
- If fewer than 3 independent reviewers can run, say so; never count a self-review as an approval.

## 6. Record
In the shiplog: each round, each reviewer's model, verdict, findings, and what you did about each. Put the final verdicts in the PR's "How it was verified" section.
