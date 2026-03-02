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

## GET `/api/auth/telegram/config`

Returns Telegram widget settings and whether auth is enabled.

Success `200` example:

```json
{
  "enabled": true,
  "botUsername": "your_bot_name",
  "requestAccess": "write"
}
```

## POST `/api/auth/telegram/callback`

Accepts Telegram Login Widget payload (`application/x-www-form-urlencoded` or JSON),
verifies signature/auth date, upserts user, returns session token and user.

Success `200` example:

```json
{
  "token": "<jwt>",
  "user": {
    "id": 42,
    "telegramId": 123456789,
    "username": "username",
    "firstName": "First",
    "lastName": "Last",
    "photoUrl": "https://...",
    "isBlocked": false,
    "isAdmin": false
  }
}
```

Errors:
- `400` invalid payload
- `401` hash mismatch / stale / invalid auth data
- `403` blocked user
- `409` replay detected

## GET `/api/me`

Returns current user from Bearer token or `gridlogger_session` cookie.

Errors:
- `401` unauthorized / invalid token
- `403` blocked user

## POST `/api/auth/logout`

Clears auth cookie and returns `{ \"status\": \"ok\" }`.
