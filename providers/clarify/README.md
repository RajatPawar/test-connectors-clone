# Clarify connector

Read-only deep connector for the Clarify CRM REST API (JSON:API envelopes).

## Auth and requests

- API key from Settings > API Keys, sent as `Authorization: api-key <key>`, not Bearer
  ([Authentication](https://developer.clarify.ai/docs/getting-started/authentication#api-keys)).
- Base URL `https://api.clarify.ai`. Every path is under `/v1/workspaces/{workspace}`; the
  workspace slug is the catalog's `workspace` metadata input, which the connector receives as
  `params.Workspace`. Proxy callers put `/v1/workspaces/<slug>/...` in their own paths.

## Objects

| Object    | Endpoint                                       | Metadata source           |
|-----------|------------------------------------------------|---------------------------|
| `person`  | `GET /v1/workspaces/{ws}/objects/person/resources`  | `GET /v1/workspaces/{ws}/schemas` |
| `company` | `GET /v1/workspaces/{ws}/objects/company/resources` | same                      |
| `deal`    | `GET /v1/workspaces/{ws}/objects/deal/resources`    | same                      |
| `meeting` | `GET /v1/workspaces/{ws}/objects/meeting/resources` | same                      |
| `task`    | `GET /v1/workspaces/{ws}/objects/task/resources`    | same                      |
| `users`   | `GET /v1/workspaces/{ws}/users`                     | static, from the List users reference |

The customer asked for "user"; the object is named `users` after its path, while the five CRM
objects are named after the `{object}` path segment Clarify uses for them.

Metadata fetches the schemas once per call and maps each `.../entities/{object}` schema's
properties (JSON Schema `type`, `format`, `enum`, `title`, `readOnly`, `xClarifyRelationship`), so
workspace custom fields are included. To-one relationship fields (`company_id`) are `reference`
fields; to-many relationships are not listed, since records only carry them with `include`.

## Record shape

Read moves `attributes` to the top level next to `id` and `type`. Values stay as Clarify returns
them: collections are objects such as `{"items": [...]}` and `name` is an object. `Raw` is untouched.

## Pagination

`page[limit]` (500, the documented maximum) and `page[offset]`, then `links.next` until it is null
([Pagination](https://developer.clarify.ai/docs/api-basics/pagination)). The live API returns
`links.next` as `http://api.clarify.ai/...`, so for every link it follows (record pages and the
cursor-paginated GET /schemas) the connector keeps the path and query and puts the base URL's
scheme and host back, so the API key never goes over plain HTTP.

## Incremental read

The five CRM objects filter on the server: `filter[_updated_at][Is after]` (Since, exclusive) and
`filter[_updated_at][Is on or before]` (Until, inclusive), RFC 3339 UTC with milliseconds
([Filtering](https://developer.clarify.ai/docs/api-basics/filtering)). Live checks confirmed the
filter and that `_updated_at` is compared below the millisecond it is returned with, so a record
shown as `.720Z` falls in the window starting at `.720Z`; consecutive windows miss nothing.
`GET /users` documents no filters, so users are read in full and filtered on `_updated_at` in
the connector.

## Rate limits

3,000 requests per minute per workspace per endpoint, rolling 60 seconds; 429 carries `Retry-After`
([Rate limits](https://developer.clarify.ai/docs/api-basics/rate-limits)).
