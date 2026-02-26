# Architecture

## Data flow

1. Device sends `POST /api/projects/{projectId}/ping` every ~30s.
2. Backend records arrival timestamp in `pings(project_id, ts)`.
3. Frontend requests availability for a window.
4. Backend computes interval status (`available` or `outage`) at 30s steps and merges adjacent segments.
5. Backend returns intervals + stats for the same window.

## Why compute in backend code

Given small scale and simple rules, interval derivation in Go is easier to evolve than SQL-heavy interval stitching.
This can move to continuous aggregates later if needed.

## Outage rule

- Configurable threshold (`OUTAGE_THRESHOLD_SECONDS`, default 120).
- For a time point `t`, status is `available` if at least one ping exists in `[t-threshold, t]`, otherwise `outage`.

## Time handling

- Storage in UTC.
- API window inputs in RFC3339.
- Frontend currently presents with `Europe/Kyiv` orientation.
