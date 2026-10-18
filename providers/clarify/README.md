# Clarify connector

Read-only deep connector for the Clarify CRM REST API (JSON:API).

## Auth and base URL

- API key from Settings > API Keys, sent as `Authorization: api-key <key>` (not `Bearer`).
- Catalog `BaseURL` is `https://api.clarify.ai`. Every path is under `/v1/workspaces/{workspace}/`;
  the workspace slug is the `workspace` metadata input and reaches the connector as
  `ConnectorParams.Workspace`. Proxy callers send the full `/v1/workspaces/<slug>/...` path.

## Objects

| Object | Endpoint | Incremental |
|---|---|---|
| `person`, `company`, `deal`, `meeting`, `task` | `GET /objects/{object}/resources` | `filter[_updated_at]`, widened to days, then trimmed |
| `user` | `GET /users` | connector-side only (no filter parameter) |

Records are flattened: `attributes.*` move to the top level next to `id` and `type`. `Raw` is the
untouched JSON:API resource.

## Metadata

`GET /schemas` once per call; each object's JSON Schema is the entry whose id ends in
`/entities/{object}`, so custom fields are included. `enum` becomes `singleSelect`, a collection
(`{items: [...]}`) of enum items `multiSelect`, `xClarifyRelationship` a `reference` to its `entity`.
The system fields (`id`, `_id`, `_created_at`, `_updated_at`, `_created_by`, `_updated_by`) are always
listed. `user` is not a schema entity, so its fields are the fixed shape `GET /users` returns.

## Pagination

`page[limit]` (max 500, used as default) and `page[offset]`; the read follows `links.next` until it is
null. The API returns `links.next` with an `http://` scheme, so the connector keeps its path and query
and takes scheme and host from the BaseURL.

## Incremental reads

`filter[_updated_at][Is after]` / `[Is on or before]` compare whole days, not instants (checked live:
a record updated at 18:36Z still matched `Is after` a later time that same day). The connector sends
dates two days wider than (Since, Until] and drops records outside the exact window by
`_updated_at` (or `_created_at` when `_updated_at` is null). Records come back ordered by `_id`, so
every page of the widened window is read.

## Rate limits

3,000 requests per minute per workspace per endpoint, rolling window; 429 with `Retry-After`.
https://developer.clarify.ai/docs/api-basics/rate-limits
