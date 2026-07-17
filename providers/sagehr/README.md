# Sage HR

Read-only connector for the [Sage HR API](https://developer.sage.com/hr/docs/v1.0.0/guides/get-started/quick-start).

## Auth

API key auth. The key is passed as-is (no prefix) in the `X-Auth-Token` header
on every request. Generated per-user from Sage HR's Settings > Integrations >
API screen (requires admin rights).

## Base URL

`https://{{.workspace}}.sage.hr/api`

Each Sage HR company has its own subdomain. `workspace` is the connector
metadata field capturing that subdomain (e.g. `mycompany` in
`https://mycompany.sage.hr`).

## Read objects

| Object | Incremental field | Notes |
|---|---|---|
| `teams` | — | no time filter documented |
| `employees` | — | no time filter documented |
| `positions` | — | no time filter documented |
| `termination-reasons` | — | no time filter documented |
| `documents/categories` | — | no pagination |
| `terminated-employees` | — | no time filter documented |
| `onboarding/categories` | — | no pagination |
| `recruitment/positions` | — | supports `status`/`hiring_manager_ids`/`group_ids`/`location_ids` filters (not wired); no time filter |
| `offboarding/categories` | — | no pagination |
| `leave-management/policies` | — | no pagination |
| `leave-management/requests` | `Since`/`Until` → `from`/`to` (date-only, `YYYY-MM-DD`) | API requires the `to`-`from` window to be under 65 days; the connector does not chunk longer requested windows |
| `leave-management/out-of-office-today` | — | provider only accepts a single `date` snapshot param, not a range, so it is not wired to `Since`/`Until` |

None of the list endpoints expose an `updated_at`-equivalent field in their
response body, so connector-side (fetch-then-discard) incremental filtering
isn't possible for the objects without provider-side time filters — see the
docs' response samples in `../../docs/openapi_spec.json`.

## Write objects

None — this connector is read-only (`Write: false`).

## Pagination

Page-number based. Paginated endpoints return `meta.next_page` (an integer, or
`null` on the last page); the connector echoes that value back as the `page`
query parameter on the next request. Endpoints with small, fixed lists (the
`*/categories` and `leave-management/policies`/`out-of-office-today` objects)
omit `meta` entirely and are read as a single page.

## Notable quirks

- Every object shares the same response envelope: `{"data": [...], "meta": {...}}`.
- Object metadata (schema) is derived by sampling one real record from each
  object's list endpoint (Sage HR has no discovery/describe endpoint).
- Rate limits are not published in the docs scraped for this connector; see
  `server/shared/limiter/defaults.go` for the corresponding TODO entry.
- Several documented GET endpoints were intentionally skipped because they
  require a parent record id in the path (e.g. `/employees/{id}/compensations`)
  or are single-record "detail" variants of an already-listed object — see the
  comment in `supports.go` for the full list and rationale.
