---
name: verify
description: Prove a change works for a real user before saying "done" or "ready". Pick 1-6 user jobs (not diff seams), run the real product like a user (browser for web apps, real commands for CLIs/APIs/libraries), check what the user sees, fix and fully retest up to 10 rounds, and report with screenshots or real output.
---
# Verify: prove the job, not the diff

## 0. Pick cases from user jobs (2 minutes)
List 1-6 things a real user does that this change touches, including one edge case (empty, error, long text, mobile width).

## 1. Plumbing checks (fast, not proof)
Lint, typecheck, unit tests. Passing them proves nothing about the user.

## 2. The real product check
- Web app: start it with the one command, drive a headless browser (Playwright), do each job click by click, screenshot each meaningful state at desktop and 390x844 mobile. Check the console and network for errors.
- CLI, API, library: run it like a user with real inputs and capture the real output.
- Performance: only if the change can affect speed; measure before and after.
Never put tokens or personal data from URLs into logs or reports. Keep auth files in .scratch/ (gitignored).

## 3. The loop (max 10 rounds)
Something wrong: fix, then rerun EVERY case, not just the failed one.

## 4. Evidence and report
Save screenshots to `.scratch/verify/<task>/`. Report per job: passed or failed, with the screenshot or output. Say plainly what you could not verify.
