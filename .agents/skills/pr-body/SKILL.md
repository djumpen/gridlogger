---
name: pr-body
description: Write or repair a PR title and description that a busy reviewer understands on a phone in 30 seconds. Plain title in the reader's words, a first sentence that tells the whole story, the product goal, measured impact, before/after proof in the medium the change happens in, then the code map, verification and risks.
---
# PR body

## 1. Title
What changes for the user, in plain words, under 60 characters. No ticket codes, no "refactor stuff".

## 2. Lede
Two or three plain sentences at the top: what was wrong, what this changes, how we know it works.

## 3. Product goal
One sentence: which user job this serves.

## 4. Problem: impact, measured
Who hits it, how often, what it costs them. Numbers if you have them.

## 5. Before / After
The proof in one place: screenshots for anything visible (commit them or upload them so they actually render in the PR), real output for logs and APIs.

## 6. What changed
Short code map: file, what changed, why.

## 7. How it was verified
Commands run, user jobs tested in the browser, code-review verdicts.

## 8. Risks and rollback
What could break, how to notice, how to revert.

## Writing rules
Short sentences. The reader's words, not internal jargon. No filler.
