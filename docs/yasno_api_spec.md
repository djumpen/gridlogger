# Yasno Outage Schedule API Spec

## Purpose

This spec describes the public Yasno outage endpoints used by the repository `denysdovhan/ha-yasno-outages` to:

1. resolve a user address to an outage `group`
2. fetch planned outage schedules for that group
3. optionally fetch probable outage schedules

This is an inferred spec from integration source code, not official vendor documentation.

## Base URL

```text
https://app.yasno.ua/api/blackout-service/public/shutdowns
```

No authentication is used by the referenced integration for these endpoints.

---

## High-level flow

### Planned outage flow

1. Fetch regions and DSOs/providers
2. Pick `regionId` and `dsoId`
3. Search street by text
4. Search house by text within chosen street
5. Resolve address to `group` + `subgroup`
6. Compose final group key as `"<group>.<subgroup>"`
7. Fetch planned outages for `regionId` and `dsoId`
8. Read schedule for the resolved group key

### Probable outage flow

1. Fetch probable outages for `regionId` and `dsoId`
2. Read slots from nested structure:
   `response[str(regionId)].dsos[str(dsoId)].groups[group].slots[weekday]`

---

## Endpoints

## 1) List regions and DSOs

### Request

```http name=http_regions.txt
GET /addresses/v2/regions
Host: app.yasno.ua
Accept: application/json
```

### Full URL

```text name=regions_url.txt
https://app.yasno.ua/api/blackout-service/public/shutdowns/addresses/v2/regions
```

### Response shape

```json name=regions_response.json
[
  {
    "id": 1,
    "value": "Київ",
    "dsos": [
      {
        "id": 11,
        "name": "ДТЕК Київські електромережі"
      }
    ]
  }
]
```

### Notes

- Response is expected to be a JSON array.
- Integration looks up region by `value`.
- Integration looks up provider/DSO by `name`.

---

## 2) Search streets

### Request

```http name=http_streets.txt
GET /addresses/v2/streets?regionId={regionId}&dsoId={dsoId}&query={query}
Host: app.yasno.ua
Accept: application/json
```

### Query parameters

| Name | Type | Required | Description |
|---|---|---:|---|
| `regionId` | integer | yes | Region identifier |
| `dsoId` | integer | yes | DSO/provider identifier |
| `query` | string | yes | Street search text |

### Full URL pattern

```text name=streets_url_pattern.txt
https://app.yasno.ua/api/blackout-service/public/shutdowns/addresses/v2/streets?regionId={regionId}&dsoId={dsoId}&query={query}
```

### Response shape

```json name=streets_response.json
[
  {
    "id": 12345,
    "name": "Хрещатик"
  }
]
```

### Notes

- Response is expected to be a JSON array.
- Consumer should let the user choose among multiple results.

---

## 3) Search houses

### Request

```http name=http_houses.txt
GET /addresses/v2/houses?regionId={regionId}&streetId={streetId}&dsoId={dsoId}&query={query}
Host: app.yasno.ua
Accept: application/json
```

### Query parameters

| Name | Type | Required | Description |
|---|---|---:|---|
| `regionId` | integer | yes | Region identifier |
| `streetId` | integer | yes | Street identifier |
| `dsoId` | integer | yes | DSO/provider identifier |
| `query` | string | yes | House search text |

### Full URL pattern

```text name=houses_url_pattern.txt
https://app.yasno.ua/api/blackout-service/public/shutdowns/addresses/v2/houses?regionId={regionId}&streetId={streetId}&dsoId={dsoId}&query={query}
```

### Response shape

```json name=houses_response.json
[
  {
    "id": 67890,
    "name": "1"
  }
]
```

### Notes

- Response is expected to be a JSON array.
- House labels may be strings, not guaranteed numeric-only.

---

## 4) Resolve outage group by address

### Request

```http name=http_group.txt
GET /addresses/v2/group?regionId={regionId}&streetId={streetId}&houseId={houseId}&dsoId={dsoId}
Host: app.yasno.ua
Accept: application/json
```

### Query parameters

| Name | Type | Required | Description |
|---|---|---:|---|
| `regionId` | integer | yes | Region identifier |
| `streetId` | integer | yes | Street identifier |
| `houseId` | integer | yes | House identifier |
| `dsoId` | integer | yes | DSO/provider identifier |

### Full URL pattern

```text name=group_url_pattern.txt
https://app.yasno.ua/api/blackout-service/public/shutdowns/addresses/v2/group?regionId={regionId}&streetId={streetId}&houseId={houseId}&dsoId={dsoId}
```

### Response shape

```json name=group_response.json
{
  "group": 3,
  "subgroup": 1
}
```

### Derived value

Construct the outage group key exactly as:

```text name=group_key.txt
{group}.{subgroup}
```

Example:

```text name=group_key_example.txt
3.1
```

### Notes

- Response is expected to be a JSON object.
- If either `group` or `subgroup` is missing, treat it as an invalid API response.

---

## 5) Planned outages by region + DSO

### Request

```http name=http_planned.txt
GET /regions/{regionId}/dsos/{dsoId}/planned-outages
Host: app.yasno.ua
Accept: application/json
```

### Full URL pattern

```text name=planned_url_pattern.txt
https://app.yasno.ua/api/blackout-service/public/shutdowns/regions/{regionId}/dsos/{dsoId}/planned-outages
```

### Response shape

Top-level object keyed by outage group string.

```json name=planned_response.json
{
  "3.1": {
    "updatedOn": "2024-11-16T21:12:00",
    "today": {
      "date": "2024-11-16T00:00:00",
      "status": "ScheduleApplies",
      "slots": [
        {
          "start": 960,
          "end": 1200,
          "type": "Definite"
        },
        {
          "start": 1200,
          "end": 1440,
          "type": "NotPlanned"
        }
      ]
    },
    "tomorrow": {
      "date": "2024-11-17T00:00:00",
      "status": "ScheduleApplies",
      "slots": [
        {
          "start": 0,
          "end": 240,
          "type": "Definite"
        }
      ]
    }
  }
}
```

### Access pattern

To get one address schedule:

1. resolve group key, e.g. `3.1`
2. fetch whole planned response for region + DSO
3. read `response["3.1"]`

### Group schedule object

| Field | Type | Required | Description |
|---|---|---:|---|
| `updatedOn` | ISO datetime string | no | Last update timestamp |
| `today` | object | no | Today's schedule |
| `tomorrow` | object | no | Tomorrow's schedule |

### Day schedule object

| Field | Type | Required | Description |
|---|---|---:|---|
| `date` | ISO datetime string | yes when day exists | Date anchor for slot conversion |
| `status` | string | no | Schedule state |
| `slots` | array | no | Time slots for the day |

### Slot object

| Field | Type | Required | Description |
|---|---|---:|---|
| `start` | integer | yes | Minutes from midnight |
| `end` | integer | yes | Minutes from midnight |
| `type` | string | yes | Slot type |

### Known `status` values

```text name=planned_status_values.txt
ScheduleApplies
WaitingForSchedule
EmergencyShutdowns
```

### Known slot `type` values

```text name=slot_type_values.txt
Definite
NotPlanned
```

### Slot semantics

- `start` and `end` are integer minutes from the start of the day.
- `1440` means midnight of the next day.
- Slot timestamps should be converted relative to the day's `date`.

### Slot conversion rule

Given:
- day date = `2024-11-16T00:00:00`
- slot = `{ "start": 960, "end": 1200, "type": "Definite" }`

Then:
- start = `2024-11-16T16:00:00`
- end = `2024-11-16T20:00:00`

Given:
- slot end = `1440`

Then:
- end = next day at `00:00:00`

---

## 6) Probable outages

### Request

```http name=http_probable.txt
GET /probable-outages?regionId={regionId}&dsoId={dsoId}
Host: app.yasno.ua
Accept: application/json
```

### Full URL pattern

```text name=probable_url_pattern.txt
https://app.yasno.ua/api/blackout-service/public/shutdowns/probable-outages?regionId={regionId}&dsoId={dsoId}
```

### Response shape

Inferred nested structure:

```json name=probable_response.json
{
  "1": {
    "dsos": {
      "11": {
        "groups": {
          "3.1": {
            "slots": {
              "0": [
                { "start": 0, "end": 240, "type": "Definite" }
              ],
              "1": [
                { "start": 480, "end": 720, "type": "Definite" }
              ]
            }
          }
        }
      }
    }
  }
}
```

### Access pattern

```text name=probable_access_pattern.txt
response[str(regionId)].dsos[str(dsoId)].groups[group].slots[str(weekday)]
```

### Weekday numbering

```text name=weekday_numbering.txt
0 = Monday
1 = Tuesday
2 = Wednesday
3 = Thursday
4 = Friday
5 = Saturday
6 = Sunday
```

### Notes

- This endpoint is optional if implementing only planned outages.
- Slot format matches planned outage slot format.

---

## Error handling

### Transport / HTTP behavior

- Expect standard HTTP errors.
- `404` should be handled distinctly if useful.
- Any non-JSON or malformed response should be treated as API failure.

### Recommended handling rules

| Condition | Recommended behavior |
|---|---|
| Network timeout | retryable error |
| 404 on lookup endpoints | "not found" result |
| Invalid response type | fail with parsing error |
| Missing `group` or `subgroup` in group response | fail with parsing error |
| Group key not found in planned response | treat as no schedule for address |

---

## Data contracts for implementation

## Contract: Region

```json name=region_contract.json
{
  "id": "integer",
  "value": "string",
  "dsos": [
    {
      "id": "integer",
      "name": "string"
    }
  ]
}
```

## Contract: Street

```json name=street_contract.json
{
  "id": "integer",
  "name": "string"
}
```

## Contract: House

```json name=house_contract.json
{
  "id": "integer",
  "name": "string"
}
```

## Contract: GroupResolution

```json name=group_resolution_contract.json
{
  "group": "integer|string",
  "subgroup": "integer|string"
}
```

## Contract: PlannedDaySlot

```json name=planned_day_slot_contract.json
{
  "start": "integer",
  "end": "integer",
  "type": "Definite|NotPlanned"
}
```

## Contract: PlannedDay

```json name=planned_day_contract.json
{
  "date": "ISO datetime string",
  "status": "string",
  "slots": [
    {
      "start": "integer",
      "end": "integer",
      "type": "Definite|NotPlanned"
    }
  ]
}
```

## Contract: PlannedGroupSchedule

```json name=planned_group_schedule_contract.json
{
  "updatedOn": "ISO datetime string",
  "today": {
    "date": "ISO datetime string",
    "status": "string",
    "slots": []
  },
  "tomorrow": {
    "date": "ISO datetime string",
    "status": "string",
    "slots": []
  }
}
```

---

## Required feature behavior for another agent

Implement a client that can:

1. fetch regions
2. select region + DSO
3. search streets by text
4. search houses by text
5. resolve address to outage group key
6. fetch planned outages
7. return normalized schedule for one group
8. convert minute-based slots into concrete datetimes
9. expose `today` and `tomorrow` schedules
10. gracefully handle missing schedules and malformed responses

### Minimum public methods

```text name=minimum_methods.txt
listRegions()
searchStreets(regionId, dsoId, query)
searchHouses(regionId, dsoId, streetId, query)
resolveGroup(regionId, dsoId, streetId, houseId)
getPlannedOutages(regionId, dsoId)
getGroupPlannedSchedule(regionId, dsoId, groupKey)
normalizeDaySchedule(dayObject)
```

### Normalized day schedule output

```json name=normalized_day_schedule.json
{
  "date": "2024-11-16",
  "status": "ScheduleApplies",
  "slots": [
    {
      "type": "Definite",
      "startMinutes": 960,
      "endMinutes": 1200,
      "startDateTime": "2024-11-16T16:00:00",
      "endDateTime": "2024-11-16T20:00:00"
    }
  ]
}
```

---

## Assumptions

- API is unofficial/inferred.
- Response fields may evolve.
- Consumers should code defensively and log raw responses when parsing fails.
- Planned outage schedules are keyed by outage group string, not directly by address.

---

## Suggested prompt for another AI agent

```text name=agent_prompt.txt
Build a client for the Yasno outage API described in yasno_api_spec.md.

Requirements:
- Implement the full planned-outage flow: regions -> streets -> houses -> group -> planned schedule.
- No auth is required.
- Treat API as unofficial and validate response shapes defensively.
- Convert slot minute ranges into absolute datetimes using the day.date field.
- Support today and tomorrow planned schedules.
- Return normalized typed objects.
- Add retries/timeouts and clear error classes.
- Keep the client independent from Home Assistant.
```