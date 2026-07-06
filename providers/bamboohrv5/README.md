# BambooHR (bamboohrv5)

Read-only connector for the BambooHR public REST API (`/api/v1`), built on the
`internal/components` framework.

## Auth

**Basic Auth**, API-key-as-username: the BambooHR API key is sent as the Basic
auth username with any non-empty password (`x`). Confirmed by the OpenAPI
spec's `basic: {type: http, scheme: basic}` security scheme, present alongside
OAuth2 on every endpoint used here. Configured via `providers.BambooHR`'s
`BasicOpts.ApiKeyAsBasic`.

## Connector metadata

- `company` — the BambooHR subdomain, e.g. `ampersand` for
  `https://ampersand.bamboohr.com`. Required; supplied by the user (there is no
  discovery endpoint or token claim to derive it from).

## Base URL

`https://{company}.bamboohr.com` — templated only on the `company` subdomain,
matching the OpenAPI spec's declared server (`https://{companyDomain}.bamboohr.com`).

## Read objects

| Object | Endpoint | Incremental field | Notes |
|---|---|---|---|
| `employees` | `GET /api/v1/employees` | none available | No `updated_at` filter on this endpoint (its filter object only exposes demographic fields like `status`, `city`, `gender`). BambooHR exposes a separate change-tracking endpoint (`GET /api/v1/employees/changed`) returning only changed employee IDs; wiring that up as a 2-step (ids → batch fetch) read is a `// TODO` in `handlers.go`. Its own doc page is a broken/404 link in the scraped docs, so its exact params are unconfirmed. |
| `jobs` | `GET /api/v1/applicant_tracking/jobs` | none available | No pagination or time filter documented; always returns the full list of non-deleted openings. |
| `applications` | `GET /api/v1/applicant_tracking/applications` | `newSince` (creation time only) | No updated-since filter is documented, so `newSince` is used as the closest available incremental signal. |
| `requests` (time off requests) | `GET /api/v1/time_off/requests` | `start`/`end` (required date-overlap window, not update time) | `ReadParams.Since`/`Until` map onto this window; defaults to the trailing 365 days when unset. |
| `schedules` | `GET /api/v1/scheduling/schedules` | `updatedAt` via OData `filter` | `updatedAt` is a documented filterable/sortable field for this endpoint. |
| `timesheet_entries` | `GET /api/v1/time_tracking/timesheet_entries` | `start`/`end` (required, must fall in the last 365 days; scopes by entry date, not update time) | Same window mapping as `requests`. |

## Pagination

- `employees`, `schedules`: cursor via `_links.next.href` — the response embeds
  a fully-qualified next-page URL that the connector follows as-is.
  `employees` uses `page[limit]` (default 250, max 2500 per spec).
- `applications`: `nextPageUrl` field in the response body, same follow-as-is pattern.
- `jobs`, `requests`, `timesheet_entries`: single page, no pagination documented.

## Notable quirks

- The `employees` id field is `employeeId`, not `id`; every other object uses `id`.
- `employees` response is wrapped in `{"data": [...], "meta": {...}, "_links": {...}}`;
  `schedules` follows the same envelope shape; `applications` uses
  `{"applications": [...], "nextPageUrl": ...}`; `jobs`, `requests`, and
  `timesheet_entries` are bare JSON arrays. Confirmed directly against
  `openapi_spec.json` response schemas/examples, not just prose docs.
- Most of the wider BambooHR API surface (compensation planning, pay grades and
  bands, alerts, total rewards, goals, onboarding, employee verification)
  requires OAuth2 per the docs; the six objects above all declare `basic` as an
  accepted security scheme, so they work with this Basic-auth connector.
- No published rate limits were found anywhere in `./docs/` (only a single,
  generic "429 - Too Many Requests" response code on an unrelated locations
  endpoint, with no policy numbers). `server/shared/limiter/defaults.go` has no
  entry for BambooHR — investigated, nothing to add beyond noting it's unpublished.

## Known blocker: live capture returns 404 for every object

A prior round's live capture (`forge_feedback/latest_capture/*.json`) hit HTTP
404 (BambooHR's marketing-site 404 HTML page, not a JSON API error) on **every**
object, using company domain `ampersand` from the test credentials file. All six
paths used exactly match the OpenAPI spec's `paths` section character-for-character
(verified independently), so this is not a connector routing bug — it indicates
`ampersand.bamboohr.com` is not a provisioned/reachable company domain for the
test account. This conclusion is corroborated by an independent, separately-built
implementation of this same connector (round `bamboohrv4`) hitting the identical
404 pattern and reaching the same conclusion after its own review. **This needs a
human to confirm/replace the test company domain** — see `forge_notes.md`.

## Out of scope this round

Write operations are intentionally not implemented (`Write: false`). Many other
GET-list endpoints exist (time off policies, who's-out, break policies/breaks,
training types, benefit coverages, locations, company files, users) that are
Basic-auth compatible per the OpenAPI spec but were not added here to keep this
first pass focused; see `forge_notes.md` for the full inventory considered.
