---
name: deep-qa
description: Whole-product QA sweep through real user jobs in a real browser (or real CLI/API calls) against a running local or staging build. Two lanes: scripted job rows that repeat every run, and blind persona journeys where an agent uses the product with only a goal. Produces a triaged bug report and a better catalog for next time.
---
# Deep QA: test the product the way people use it

## Inputs
Target URL, build, accounts, mode: QUICK (spine only), CORE (spine + 2 personas), FULL (spine + 4 personas + mobile).

## Phase 0: Preflight
App runs, accounts work, browser works.

## Phase 1: The job catalog
`.scratch/qa/catalog.md`: one row per user job with steps and the expected result. Create it from the product if missing.

## Phase 2: Plan
Pick rows and personas for this mode.

## Phase 3: Execute
- 3a. Spine rows: run each job in the browser, screenshot, pass or fail.
- 3b. Persona journeys: a sub-agent gets only a persona and a goal ("new user who wants to add 3 items and share them"), no instructions. It records where it got stuck.
- 3c. Judge and triage yourself, never delegated: real bug, UX friction, or not a problem. Severity by user impact.

## Phase 4: Trust gate
Plant or recall one known bug and check the run catches it. A run that cannot say no is not evidence.

## Phase 5: Report
Bugs ranked by severity, each with steps, screenshot and expected vs actual. Then friction notes.

## Phase 6: Make the next run better
Add new rows for every bug found. Remove rows that never fail and test nothing.

## Rails
Never test against production data. Never create real payments or emails.
