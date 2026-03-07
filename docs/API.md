# API contract

Base path: `/api`

## GET `/api/projects`

Returns public list of projects for landing page.

Success `200` example:

```json
{
  "projects": [
    {
      "id": 1,
      "name": "Саксаганського 12А",
      "slug": "saksa-12a",
      "userId": 42,
      "city": "Київ",
      "description": "Ввод #1",
      "isPublic": true,
      "hasOutageSchedule": true,
      "createdAt": "2026-03-02T10:00:00Z"
    }
  ]
}
```

## GET `/api/project-slugs/{slug}`

Returns a single project by slug.

Success `200` example:

```json
{
  "project": {
    "id": 1,
    "name": "Саксаганського 12А",
    "slug": "saksa-12a",
    "userId": 42,
    "city": "Київ",
    "description": "Ввод #1",
    "isPublic": true,
    "hasOutageSchedule": true,
    "createdAt": "2026-03-02T10:00:00Z"
  }
}
```

## GET `/api/settings`

Returns authenticated user project list for settings page.

Errors:
- `401` unauthorized
- `403` blocked user

## POST `/api/settings/projects`

Creates a new project for authenticated user.

Request JSON:

```json
{
  "name": "Лесі Українки 8Б",
  "city": "Київ",
  "slug": "lesi-8b",
  "isPublic": true
}
```

Success `201` example:

```json
{
  "project": {
    "id": 12,
    "name": "Лесі Українки 8Б",
    "slug": "lesi-8b",
    "city": "Київ"
  },
  "redirectTo": "/a/settings/project/12"
}
```

Errors:
- `400` invalid fields / slug format
- `401` unauthorized
- `403` blocked user
- `409` slug already exists

Slug rules:
- at least 3 chars
- only `a-z`, `0-9`, `-`
- reserved value `api` is not allowed

## GET `/api/settings/projects/{projectId}`

Returns owner project details (including project `secret`).

Errors:
- `401` unauthorized
- `403` forbidden (not owner)
- `404` not found

## POST `/api/settings/projects/{projectId}`

Updates owner project fields (`name`, `city`, `slug`, `isPublic`).

Errors:
- `400` invalid fields / slug format
- `401` unauthorized
- `403` forbidden (not owner)
- `404` not found
- `409` slug already exists

## GET `/api/settings/projects/{projectId}/yasno`

Returns saved Yasno group configuration for this project and the latest schedule preview if the upstream fetch succeeds.

Success `200` example:

```json
{
  "config": {
    "projectId": 1,
    "regionId": 1,
    "regionName": "Київ",
    "dsoId": 11,
    "dsoName": "ДТЕК Київські електромережі",
    "streetId": 12345,
    "streetName": "вул. Саксаганського",
    "houseId": 67890,
    "houseName": "12/А",
    "group": "3.1",
    "createdAt": "2026-03-07T10:00:00Z",
    "updatedAt": "2026-03-07T10:05:00Z"
  },
  "schedule": {
    "group": "3.1",
    "address": "вул. Саксаганського, 12/А",
    "updatedAt": "2026-03-07T08:15:00+02:00",
    "days": [
      {
        "key": "today",
        "label": "Сьогодні",
        "weekdayShort": "Сб",
        "date": "2026-03-07",
        "status": "ScheduleApplies",
        "slots": [
          { "startMinute": 480, "endMinute": 600, "type": "Definite" }
        ]
      }
    ]
  },
  "scheduleError": ""
}
```

Errors:
- `400` invalid project id
- `401` unauthorized
- `403` forbidden
- `404` project not found
- `502` Yasno upstream error

## GET `/api/settings/projects/{projectId}/yasno/regions`

Returns Yasno regions with DSOs/providers for the address selection flow.

Errors:
- `400` invalid project id
- `401` unauthorized
- `403` forbidden
- `404` project not found
- `502` Yasno upstream error

## GET `/api/settings/projects/{projectId}/yasno/streets?regionId=<id>&dsoId=<id>&query=<text>`

Searches Yasno streets for the selected region and DSO.

Errors:
- `400` invalid params
- `401` unauthorized
- `403` forbidden
- `404` project not found / Yasno lookup not found
- `502` Yasno upstream error

## GET `/api/settings/projects/{projectId}/yasno/houses?regionId=<id>&streetId=<id>&dsoId=<id>&query=<text>`

Searches Yasno houses for the selected street.

Errors:
- `400` invalid params
- `401` unauthorized
- `403` forbidden
- `404` project not found / Yasno lookup not found
- `502` Yasno upstream error

## POST `/api/settings/projects/{projectId}/yasno/preview`

Resolves the Yasno group for the selected address and returns the current schedule preview without saving it.

Request JSON:

```json
{
  "regionId": 1,
  "regionName": "Київ",
  "dsoId": 11,
  "dsoName": "ДТЕК Київські електромережі",
  "streetId": 12345,
  "streetName": "вул. Саксаганського",
  "houseId": 67890,
  "houseName": "12/А"
}
```

Errors:
- `400` invalid payload
- `401` unauthorized
- `403` forbidden
- `404` project not found / Yasno lookup not found / schedule not found for this group
- `502` Yasno upstream error

## POST `/api/settings/projects/{projectId}/yasno`

Resolves and saves Yasno identifiers for this project in `dtek_groups`.

Request JSON: same as preview endpoint.

Success `200` body: same shape as the preview endpoint, but `config` includes persisted timestamps.

Errors:
- `400` invalid payload
- `401` unauthorized
- `403` forbidden
- `404` project not found / Yasno lookup not found / schedule not found for this group
- `502` Yasno upstream error

## DELETE `/api/settings/projects/{projectId}/yasno`

Removes the saved Yasno group binding from the current project.

Errors:
- `400` invalid project id
- `401` unauthorized
- `403` forbidden
- `404` project not found

## GET `/api/projects/{projectId}/yasno`

Returns the public Yasno planned schedule for a configured project.

Success `200` example:

```json
{
  "schedule": {
    "group": "3.1",
    "address": "вул. Саксаганського, 12/А",
    "updatedAt": "2026-03-07T08:15:00+02:00",
    "days": [
      {
        "key": "today",
        "label": "Сьогодні",
        "weekdayShort": "Сб",
        "date": "2026-03-07",
        "status": "ScheduleApplies",
        "slots": [
          { "startMinute": 480, "endMinute": 600, "type": "Definite" }
        ]
      }
    ]
  }
}
```

Errors:
- `400` invalid project id
- `404` project not found / Yasno is not configured / schedule not found for this group
- `502` Yasno upstream error

## GET `/api/settings/projects/{projectId}/telegram-bot/groups`

Returns Telegram groups linked to the current project through virtual Telegram users owned by the authenticated user.

Success `200` example:

```json
{
  "botUsername": "svitlohomes_bot",
  "groups": [
    {
      "virtualUserId": 91,
      "telegramId": -1001234567890,
      "title": "Світло ЖК Сонце",
      "chatType": "supergroup",
      "username": "",
      "addedAt": "2026-03-07T10:00:00Z"
    }
  ]
}
```

Errors:
- `401` unauthorized
- `403` forbidden (not owner)
- `404` project not found

## POST `/api/settings/projects/{projectId}/telegram-bot/groups`

Finds a Telegram group by full title in bot `getUpdates`, creates or reuses a virtual Telegram user, and subscribes it to the current project notifications.

Request JSON:

```json
{
  "title": "Світло ЖК Сонце"
}
```

Success `201` example:

```json
{
  "group": {
    "virtualUserId": 91,
    "telegramId": -1001234567890,
    "title": "Світло ЖК Сонце",
    "chatType": "supergroup",
    "username": "",
    "addedAt": "2026-03-07T10:00:00Z"
  }
}
```

Errors:
- `400` invalid payload
- `401` unauthorized
- `403` forbidden (not owner)
- `404` project not found / bot has not seen this group yet
- `409` same group title resolves to multiple chats / group belongs to another owner
- `503` telegram bot is not configured

## DELETE `/api/settings/projects/{projectId}/telegram-bot/groups/{virtualUserId}`

Removes the linked Telegram group from the current project. If that virtual user is not used by any other project, backend also deletes the virtual user and Telegram chat record.

Errors:
- `400` invalid path params
- `401` unauthorized
- `403` forbidden (not owner)
- `404` project or linked group not found

## GET `/api/projects/{projectId}/notifications/subscription`

Returns current authenticated user subscription for Telegram status notifications.

Response example:

```json
{
  "subscribed": true
}
```

Errors:
- `401` unauthorized
- `403` blocked user
- `404` project not found

## POST `/api/projects/{projectId}/notifications/subscription`

Creates or updates authenticated user subscription.

Request JSON:

```json
{
  "subscribed": true
}
```

Response example:

```json
{
  "subscribed": true
}
```

Errors:
- `400` invalid payload
- `401` unauthorized
- `403` blocked user
- `404` project not found

## POST `/api/projects/{projectId}/ping`

Records a ping using server arrival timestamp.

Required header:
- `X-Project-Secret`

Success:
- `204 No Content`

Errors:
- `401` missing/invalid secret
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
