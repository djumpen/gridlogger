---
name: investigate
description: Root-cause debugging for any bug, regression, flaky test or unexplained failure. Evidence first, one testable hypothesis, a fix at the cause, and a regression test that fails without the fix. Use when the user reports a bug, a test flakes, or something "randomly" breaks.
---
# Investigate: root cause before fix

## What counts as a symptom patch
A retry, a try/catch that hides the error, a timeout bump, a null check without knowing why it is null. Not allowed unless the cause is outside our code, and then say so.

## Phase 1: Evidence
Reproduce it. Record exact steps, input, error, logs, versions. No reproduction means no fix yet: gather more evidence.

## Phase 2: Pattern check
When did it start (git log, bisect)? What else uses the same code? Does it happen everywhere or only in one case?

## Phase 3: Test the hypothesis
Write ONE hypothesis. Design a check that would prove it wrong. Run it. If wrong, write the next one. Two failed hypotheses with the same idea means step back and gather more evidence.

## Phase 4: Fix and regression test
Write the test first and watch it fail. Fix at the cause. Watch the test pass. Run the full suite and the verify skill.

## Phase 5: Report
Symptom, root cause (file:line), why it happened, the fix, the test, and other places with the same risk.
