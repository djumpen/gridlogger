---
name: garden-codebase
description: Periodic, evidence-based codebase maintenance that finds and completes ONE bounded cleanup without changing product behavior: proven dead-code removal, hot-file extraction, stable duplication consolidation, or moving code to where it belongs. Use for "clean up", "tech debt", or a scheduled weekly gardening run.
---
# Garden the codebase

## Modes
SCAN (report candidates only) or FIX (complete one slice as a PR via ship-loop).

## 1. Set the boundary
No behavior change. No public API change. One slice per run.

## 2. Find evidence, not taste
- Large files: over 500 lines, or changed in many recent commits (git log). Tune in AGENTS.md.
- Dead code: proven unused by search across the repo, configs and dynamic imports, plus a passing build after removal.
- Duplication: the same logic in 3+ places that has stayed stable.

## 3. Pick exactly one slice
Highest value, lowest risk. Write why.

## 4. Pin what must not change
Add or confirm tests that cover the current behavior before touching anything.

## 5. Make the smallest complete change
Finish the slice fully. No half migrations.

## 6. Verify
Full tests, build, and the verify skill on the affected user jobs.

## 7. Report and hand off
What changed, the evidence, lines removed, and the next 3 candidates.
