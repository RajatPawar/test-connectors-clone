# BambooHR (bamboohrv4)

Read-only connector for the BambooHR public REST API (`/api/v1`), built on the
`internal/components` framework.

## Auth

**Basic Auth**, API-key-as-username: the BambooHR API key is sent as the Basic
auth username with any non-empty password (`x`). Configured via
`providers.BambooHR`'s `BasicOpts.ApiKeyAsBasic`.

## Connector metadata

- `company` — the BambooHR subdomain, e.g. `ampersand` for
  `https://ampersand.bamboohr.com`. Required; supplied by the user (there is no
  discovery endpoint or token claim to derive it from).

## Base URL

`https://{company}.bamboohr.com` — templated only on the `company` subdomain,
matching the OpenAPI spec's declared server.

## Read objects

| Object | Endpoint | Incremental field | Notes |
|---|---|---|---|
| `employees` | `GET /api/v1/employees` | none available | No `updated_at` filter on this endpoint. BambooHR exposes a separate change-tracking endpoint (`GET /api/v1/employees/changed`) returning only changed employee IDs; wiring that up as a 2-step (ids → batch fetch) read is a `// TODO` in `handlers.go`, out of scope for this pass. |
| `jobs` | `GET /api/v1/applicant_tracking/jobs` | none available | No pagination or time filter documented; always returns the full list of non-deleted openings. |
| `applications` | `GET /api/v1/applicant_tracking/applications` | `newSince` (creation time only) | No updated-since filter is documented, so `newSince` is used as the closest available incremental signal. |
| `requests` (time off requests) | `GET /api/v1/time_off/requests` | `start`/`end` (required date-overlap window, not update time) | `ReadParams.Since`/`Until` map onto this window; defaults to the trailing 365 days when unset. |
| `schedules` | `GET /api/v1/scheduling/schedules` | `updatedAt` via OData `filter` | Only incremental field documented for this endpoint. |
| `timesheet_entries` | `GET /api/v1/time_tracking/timesheet_entries` | `start`/`end` (required, must fall in the last 365 days; scopes by entry date, not update time) | Same window mapping as `requests`. |

## Pagination

- `employees`, `schedules`: cursor via `_links.next.href` — the response embeds
  a fully-qualified next-page URL that the connector follows as-is.
- `applications`: `nextPageUrl` field in the response body, same pattern.
- `jobs`, `requests`, `timesheet_entries`: single page, no pagination
  documented.

## Notable quirks

- The `employees` id field is `employeeId`, not `id`; every other object uses
  `id`.
- `employees` response is wrapped in `{"data": [...], "meta": {...}, "_links": {...}}`;
  `schedules` follows the same envelope shape; `applications` uses
  `{"applications": [...], "nextPageUrl": ...}`; `jobs`, `requests`, and
  `timesheet_entries` are bare JSON arrays.
- Most of the wider BambooHR API surface (compensation planning, pay grades and
  bands, alerts, total rewards, goals endpoint aggregates, onboarding, employee
  verification) explicitly requires **OAuth2** per the docs and is out of scope
  for this Basic-auth connector.
- No published rate limits were found in the docs (`./docs/`); the repository
  checked for a `server/shared/limiter/defaults.go` file to register them but it
  does not exist in this checkout.

## Out of scope this round

Write operations are intentionally not implemented (`Write: false`). Many other
GET-list endpoints exist (time off policies, who's-out, break policies/breaks,
training types, benefit coverages, locations, company files, users) that are
Basic-auth compatible per the OpenAPI spec but were not added here to keep this
first pass focused; see `forge_notes.md` for the full inventory considered.
