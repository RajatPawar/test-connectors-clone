# ActiveCampaign Connector

Read-only connector for the ActiveCampaign v3 REST API.

## Auth

API key auth via the `Api-Token` HTTP header (per-user key from
Settings > Developer). No OAuth, no Bearer/Basic.

## Base URL

`https://{workspace}.api-us1.com` — per-tenant subdomain (`workspace`
connector metadata field, `DefaultValue: "youraccountname"`). `/api/3` is
appended by the request builders, not baked into the catalog `BaseURL`. Docs
warn `api-us1.com` isn't guaranteed for every account — real API URL comes
from My Settings > Developer.

## Read objects

Object names are copied verbatim from the v3 REST URL path segments (e.g.
pipelines live at `/dealGroups`, not `/pipelines`):

| Object | Incremental field | Strategy |
|---|---|---|
| `contacts` | `udate` | Native `filters[updated_after]`/`filters[updated_before]` |
| `deals` | `mdate` | Native `filters[updated_after]`/`filters[updated_before]` |
| `accounts` | `updatedTimestamp` | Client-side (no query filter param exists) |
| `campaigns` | `mdate` | Client-side (no query filter param exists) |
| `lists` | `udate` | Client-side (no query filter param exists) |
| `dealGroups` | `udate` | Client-side (no query filter param exists) |
| `users` | — | Full sync only — no time field in the response at all |
| `tags` | — | Full sync only — only `cdate` exists; never used as an updated-at stand-in |

Client-side means every page is fetched and out-of-window records are
discarded in `parseReadResponse` (`readhelper.MakeTimeFilterFunc`,
`Unordered`, since none of these four document a sortable time field).

## Pagination

Offset-based: `limit` (default 100) + `offset` (zero-based). The response
never echoes the requested offset back, so the next offset is derived from
the outgoing request's own URL, stopping once it reaches `meta.total` (a
sibling of the top-level plural collection key — sometimes a JSON string).

## Notable quirks

- **Envelope**: every object nests records under a key identical to the
  object name, e.g. `{"deals": [...], "meta": {"total": "2"}}` — verified
  per-object against `openapi_spec.json`, not assumed from one.
- `dealGroups` responses also carry a sibling `dealStages` array; the
  records extractor only reads the `dealGroups` key.
- **Metadata source split**: `deals`, `campaigns`, `users`, `lists`, `tags`
  resolve from the embedded `metadata/schemas.json`. `contacts`, `accounts`,
  `dealGroups` are missing from it (their OpenAPI responses only declare
  free-text `examples`, no JSON Schema) and fall back to live response
  sampling via `schema.NewCompositeSchemaProvider`.
- Deal reads enforce pipeline permissions: an unauthorized pipeline returns
  only `id`/`title`/`isDisabled=1`, not an error.
- Rate limit: 5 requests/second/account (429 + `Retry-After`;
  `RateLimit-Limit`/`RateLimit-Remaining` headers).
- Scalar values (ids, `meta.total`) are often JSON strings, not numbers —
  parsed defensively via `jsonquery.TextWithDefault`.
- Not implemented: writes, and objects outside the PM's priority list
  (`dealStages`, `dealTasks`, ecommerce objects — the latter need a
  connected store and are typically empty in a sandbox account).
