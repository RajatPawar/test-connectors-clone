# Clarify connector

Read-only deep connector for the Clarify CRM REST API (JSON:API envelopes).

## Auth and requests

- API key from Settings > API Keys, sent as `Authorization: api-key <key>` (not `Bearer`)
  ([Authentication](https://developer.clarify.ai/docs/getting-started/authentication#api-keys)).
  Clarify also offers OAuth 2.0 for partner apps; it is not modelled here.
- Base URL `https://api.clarify.ai`; every path is `/v1/workspaces/{workspace}/...`. The
  workspace slug is the catalog's `workspace` metadata input, which reaches the connector as
  `ConnectorParams.Workspace`. Proxy callers put the slug in their own paths.

## Objects

| Object    | Endpoint                                         | Metadata                         |
|-----------|--------------------------------------------------|----------------------------------|
| `person`  | `GET /v1/workspaces/{ws}/objects/person/resources`  | `GET /v1/workspaces/{ws}/schemas` |
| `company` | `GET /v1/workspaces/{ws}/objects/company/resources` | same                              |
| `deal`    | `GET /v1/workspaces/{ws}/objects/deal/resources`    | same                              |
| `meeting` | `GET /v1/workspaces/{ws}/objects/meeting/resources` | same                              |
| `task`    | `GET /v1/workspaces/{ws}/objects/task/resources`    | same                              |
| `users`   | `GET /v1/workspaces/{ws}/users`                     | static, from the users reference  |

Names are the URL path segments, so the plan's "user" is `users`. Custom objects (`c_*`) use
the same endpoints but were not requested.

The schemas endpoint returns every schema in one call as JSON Schema; built-in objects are the
`.../schemas/entities/{object}` entries (fetched once per `ListObjectMetadata` call). Enums become
select values, `format: date|date-time` become date/datetime, single foreign-key relationships
(`company_id`) become references. The other side of a relationship (a deal's people) is not in a
record's `attributes`, so it is left out of the metadata.

## Record shape

Read lifts `attributes` to the top level next to `id` and `type`; `Raw` is untouched. Nested values
(`name`, collection fields such as `email_addresses: {"items": [...]}`) stay as objects.

## Pagination

`page[limit]` (500, the documented maximum) and the response's `links.next`, null on the last page
([Pagination](https://developer.clarify.ai/docs/api-basics/pagination)). The API returns the next
link with an `http://` scheme, so the connector keeps its path and query but uses the scheme and
host of the request it just made.

## Incremental read

The resource objects filter on `_updated_at` with `filter[_updated_at][Is after]` and
`[Is before]` ([Filtering](https://developer.clarify.ai/docs/api-basics/filtering)). Live, the
filter compared whole seconds although timestamps carry milliseconds, so the request asks for a
window widened to whole seconds plus one second each side, and the connector trims records to
`(Since, Until]`. `/users` documents no filter: users are read in full and filtered the same way.

## Rate limits

3,000 requests per minute per workspace per endpoint, rolling window; 429 with `Retry-After`
([Rate limits](https://developer.clarify.ai/docs/api-basics/rate-limits)).
