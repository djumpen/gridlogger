---
name: cto-office
description: Plan before you build. Decide whether a request fits ONE focused PR (write a self-contained brief for ship-loop) or needs a multi-PR train (parallel research, a reviewed plan with an explicit "NOT building" fence, and a launch prompt for cto-exec). Use for any request bigger than a one-line fix, or when the user says "plan", "architect", "how should we build".
---
# CTO Office: plan the smallest thing that safely ships

## Mindset
- The product goal decides everything. Write it in one sentence a user would recognize.
- The smallest plan that reaches the goal wins. Every extra PR needs a reason.
- Say what you are NOT building. That list protects the goal.

## Stop rules
Ask the human only for: new dependencies, public API or schema changes, migrations, secrets, user data, production. AGENTS.md rules win if stricter.

## State
Keep the plan in `.scratch/plans/<task>.md`. Update it; never keep the plan only in chat.

## Phase 0: Intake and decide the shape
- Restate the goal, the user, and how we will know it worked.
- If it fits one PR (one reviewable diff, one user job): write a ship-loop brief with goal, scope, NOT building, files likely touched, how to verify. Stop here.
- Otherwise continue to a PR train.

## Phase 1: Research (parallel, read-only)
Start 2-4 read-only sub-agents: code map of the affected area, existing patterns to reuse, risks (data, auth, performance), and how it is tested today. Each returns facts with file:line, not opinions.

## Phase 2: Write the plan
Sections: Goal, Users and jobs, NOT building, Current state (with file:line), Design, Risks and rollback, Verification per user job, Open questions for the human.

## Phase 3: The PR train
Split into PRs that each ship something reviewable and safe on its own. For each: title, scope, depends on, how to verify, size estimate. Prefer independent PRs; stack only when needed.

## Phase 4: Review the plan
Two fresh-context reviewers (different models if available) check: does it reach the goal, is anything missing, is anything unnecessary, is the order safe. Fix and record what changed.

## Phase 5: Present
Give the human: the goal, the PR list, NOT building, open questions, and a launch prompt that starts cto-exec in a fresh session. Pasting that prompt is the approval.

## Lessons
After delivery, add one line to `.scratch/lessons.md` about what the plan got wrong.
