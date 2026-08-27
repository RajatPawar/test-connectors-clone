# ActiveCampaign Connector

Read-only connector for the [ActiveCampaign v3 API](https://developers.activecampaign.com/reference).

## Auth

API key, sent in the `Api-Token` header. Found per-user under **Settings >
Developer**.

## Base URL

Per-tenant subdomain: `https://{workspace}.api-us1.com`. `workspace` is the
account subdomain (e.g. `mycompany` in `https://mycompany.api-us1.com`). The
docs explicitly warn that `api-us1.com` is not guaranteed for every account
and that the full API URL under **My Settings > Developer** is the real
source of truth — but the OpenAPI spec's `servers` block only declares the
`{youraccountname}.api-us1.com` template, so that's what this connector
follows (see CLAUDE.md's "spec is LAW" rule). Every request path is
`{BaseURL}/api/3/{object}`.

## Read objects

`contacts`, `deals`, `accounts`, `lists`, `campaigns`, `tags`, `users`,
`dealTasks`. For every one of these the URL path segment, the response
envelope key, and the object name are identical (e.g. `GET /api/3/deals` ->
`{"deals": [...], "meta": {...}}`).

**Incremental sync** (filters on `updated_at`-equivalent field; per-object,
names differ — copied verbatim from docs/openapi_spec.json):

| Object | Strategy | Field |
|---|---|---|
| `contacts` | native `filters[updated_after]`/`filters[updated_before]` query params | `udate` |
| `deals` | native `filters[updated_after]`/`filters[updated_before]` query params | `mdate` |
| `accounts` | connector-side filter (no query param exists) | `updatedTimestamp` (camelCase) |
| `lists` | connector-side filter (no query param exists) | `udate` |
| `campaigns` | connector-side filter (no query param exists) | `mdate` |
| `dealTasks` | connector-side filter (no query param exists) | `udate` |
| `tags` | **full read only** — no update-time field exists on the record at all (only `cdate`) | n/a |
| `users` | **full read only** — no timestamp field exists on the record at all | n/a |

Connector-side filtered objects are fetched unfiltered from the API and
filtered locally in `parse.go` (`readhelper.MakeTimeFilterFunc`, `Unordered`
mode — none of these endpoints document an `orders[]` option for their
timestamp field, so no early-stop optimization is applied; every page is
fetched and filtered).

## Pagination

Offset/limit, no cursor. `limit` (max 100) + `offset` (zero-based), looping
`offset += limit` until `offset >= meta.total`. `meta.total`'s JSON type is
inconsistent across endpoints (int in prose docs, string in the OpenAPI
schema), so it's parsed defensively as text
(`jsonquery.TextWithDefault`). `NextPage` carries the full next request URL
(cloned from the current request, with `offset` bumped), so any query
parameters set on the first request — including `filters[updated_after]` —
are preserved across pages.

## Metadata (schema)

Priority 1 (`docs/schemas.json`, embedded in `metadata/schemas.json`) covers
`deals`, `lists`, `campaigns`, `tags`, `users`, `dealTasks` — their OpenAPI
`200` responses declare a JSON Schema. `contacts` and `accounts` don't: their
spec responses only carry bare JSON examples, no schema, so those two fall
back to live response sampling (`schema.NewObjectSchemaProvider`, Priority 3)
via `CompositeSchemaProvider`.

## Not implemented this round

- Write support (out of scope for this round; `Support.Write = false`).
- Everything outside the 8 objects above (e.g. `automations`, `forms`,
  `ecomCustomers`/`ecomOrders`, `customObjects`, browse session/GraphQL
  ecommerce APIs, WhatsApp, AI customizations, agency/reseller endpoints,
  Segments API) — either action/verb endpoints, discovery endpoints, objects
  needing store/product-catalog setup not present in a plain sandbox, or
  simply out of scope for the priority object list.

## Rate limits

5 requests/second per account, shared with the eComm GraphQL API.
https://developers.activecampaign.com/reference/rate-limits
