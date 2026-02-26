# API contract

Base path: `/api`

## POST `/api/projects/{projectId}/ping`

Records a ping using server arrival timestamp.

Success:
- `204 No Content`

Errors:
- `400` invalid `projectId`

## GET `/api/projects/{projectId}/availability?from=<RFC3339>&to=<RFC3339>`

Returns intervals and totals within the requested visible window.

Success `200` body shape:

```json
{
  "projectId": 1,
  "from": "2026-02-25T00:00:00Z",
  "to": "2026-03-03T00:00:00Z",
  "intervals": [
    {"start":"...","end":"...","status":"available"},
    {"start":"...","end":"...","status":"outage"}
  ],
  "stats": {
    "availabilityPercent": 97.5,
    "totalAvailableHours": 163.8,
    "totalOutageHours": 4.2
  },
  "timezone": "Europe/Kyiv",
  "sampleEvery": "30s"
}
```

Errors:
- `400` invalid path params or query window
- `404` unknown route/method
