# Otter.ai Connector

Read-only connector for the [Otter.ai Public API](https://help.otter.ai/hc/en-us/articles/36130822688279-Otter-ai-Public-API).

## Auth

API key auth: Bearer token in the `Authorization` header —
`Authorization: Bearer YOUR_API_KEY`. Keys are created in the Otter UI
(Integrations → Developer) and are only available on **Enterprise workspaces**; a
non-Enterprise account cannot create a key or use this API at all.

## Base URL

`https://api.otter.ai`. All documented endpoints live under `/v1`; the version
segment is applied per-request rather than baked into the catalog `BaseURL`.
No per-tenant/region template variable exists.

## Read objects

| Object | Endpoint(s) | Notes |
|---|---|---|
| `conversations` | `GET /v1/conversations` | Cursor-paginated. |
| `channels` | `GET /v1/channels` | Single page (no pagination documented). |
| `channels/members` | `GET /v1/channels` + `GET /v1/channels/{id}/members` | See quirks. |
| `workspace` | `GET /v1/workspace` | Single object, wrapped as one row. |

**Incremental sync:** not implemented. No object documents an
`updated_at`/`updated_since` filter — `conversations` only exposes `created_at`
(creation, not modification time). `ReadParams.Since`/`Until` are intentionally
ignored rather than faked against `created_at`. Otter recommends
[workspace webhooks](https://help.otter.ai/articles/10924400) instead of polling.

**Out of scope:** `POST /conversations` (a write) and
`GET /conversations/{id}/audio` (a download-link, not a data object). Also
deferred: `include=...` relationship data on `GET /conversations/{id}`
(`action_items`, `insights`, `outline`, `transcript`, `custom_prompt`) — no
bulk endpoint exists, so exposing these needs an N+1 fan-out. See TODO in
`supports.go`.

## Pagination

Cursor-based, but only `GET /conversations` documents it: `meta.has_more` +
`meta.next_cursor`, requested via `?cursor=<next_cursor>`. `GET /channels`,
`GET /channels/{id}/members`, and `GET /workspace` show only
`meta.retrieved_at` in their examples, so those are treated as single-page.

## Notable quirks

- **Response envelope:** every response wraps its payload under a top-level
  `data` field (array for lists, single object for `GET /workspace`) plus a
  `meta` object. Pagination lives in `meta`, never alongside the records.
- **`channels/members` has no bulk endpoint.** `GET /channels/{id}/members`
  takes one channel ID at a time, so this connector fetches the channel list,
  then issues one sequential members call per channel (staying within the
  10 req/sec limit) and concatenates every roster into one `Read()`.
- **Enterprise-only:** every endpoint requires an Enterprise workspace API
  key. A 401/403, or `GET /workspace` 404 ("User not in a workspace"), likely
  signals an account entitlement gap, not a connector bug.
- **Metadata (schema):** no discovery/describe endpoint — schemas are derived
  by sampling one record from each object's response.
- Rate limit: 10 req/sec (Enterprise), `429` on excess. Not yet added to
  `server/shared/limiter/defaults.go` (not part of this checkout).
