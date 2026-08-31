# ActiveCampaign connector

Read-only connector for the ActiveCampaign v3 REST API.

## Auth
- **Type**: API key sent as an HTTP header.
- **Header**: `Api-Token: <per-user key>` (from *My Settings → Developer*).

## Base URL
- Per-tenant: `https://{{.workspace}}.api-us1.com` where `workspace` is the
  account name (a connection-time metadata input).
- The connector appends the `api/3` path prefix to every request.
- Note: the docs warn `api-us1.com` is not guaranteed for every account; the
  full API URL should be taken from the Developer tab. Here it is templated via
  the `workspace` metadata variable.

## Read objects
- `contacts`, `deals`, `accounts`, `campaigns`, `lists`, `tags`, `users`

### Incremental sync
- `contacts` and `deals` support `filters[updated_after]` /
  `filters[updated_before]`, wired to `Since` / `Until`.
- `accounts`, `campaigns`, `lists`, `tags` and `users` have **no** updated-at
  filter and are always read in full (a documented API limitation).

## Pagination
- Offset-based: `limit` (100 per page) + zero-based `offset`.
- `meta.total` (sibling of the collection key) drives when to stop. The current
  offset is derived from the request URL because the response does not echo it.

## Notable quirks
- List responses nest the collection under a **top-level plural key** named
  after the object (e.g. `{"contacts": [...], "meta": {...}}`), never a
  top-level array.
- Scalar ids and counts (including `meta.total`) are frequently returned as JSON
  strings, so `meta.total` is parsed defensively.
- Rate limit: 5 requests/second per account.
- Write operations exist in the API but are out of scope for this connector.
