---
name: cto-exec
description: Execute an approved cto-office plan as a PR train in a fresh session. Pasting the launch prompt is the approval; no second ask. One implementer per PR in its own worktree, each PR runs the ship-loop discipline, the lead reviews every PR, keeps the stack healthy, re-plans when reality disagrees, and delivers all PRs merge-ready.
---
# CTO Exec: run the PR train

## The product goal owns completion
Done means the user job from the plan works in the real product, not that every PR exists.

## Mindset
You are the lead, not the typist. Brief, review, unblock, integrate.

## State: the execlog
`.scratch/exec/<task>.md`: PR list with status (todo, building, in review, ready, blocked), branch, PR link, last verification, open findings. Update after every event so a restart can resume.

## Phase 1: Setup
Read the plan. Create one branch per PR (stacked only where the plan says so). Confirm the one-command run works on the base.

## Phase 2: Run the train
1. Brief each implementer (one agent per PR, its own git worktree): goal, scope, NOT building, files, verification, and "follow the ship-loop skill".
2. Every PR runs ship-loop: plan, implement, verify in the browser, 3 local reviews with code-review, draft PR via pr-body.
3. Review each PR yourself against the plan: does it do its slice and nothing more?
4. Keep the stack healthy: rebase children when a parent changes, rerun their verification.
Real-product rule: no PR is ready without screenshots or real output from the running app.

## Re-plan protocol
If a PR shows the plan is wrong, stop that branch, update the plan with the reason, get one reviewer to check the change, then continue. Scope growth beyond the plan goes to the human.

## Phase 3: Integrate and final gates
Merge all branches locally into an integration branch, run the full test suite and the verify skill on every user job in the plan.

## Phase 4: Deliver and report
To the human: PR links in merge order, what each one does in one line, proof screenshots, anything blocked and why. Never merge yourself.

## Communication
Short status at each PR milestone. Bad news first.
