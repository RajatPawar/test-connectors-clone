# ConnectWise PSA (Manage) Connector

Deep connector for the ConnectWise PSA REST API (codebase `apis/3.0`), built on
`internal/components`.

## Auth

HTTP **Basic auth**. The username is `companyId+publicKey` and the password is
`privateKey` (base64-encoded by the transport). Credentials are collected by the
Ampersand platform; this connector does not build the header itself.

Every request also sends a `clientId` header — a per-integration GUID required by
ConnectWise on all API calls. It is supplied as connector metadata and is stable
for the lifetime of a connection.

## Connector metadata

- `workspace` — the regional PSA host, e.g. `api-na.myconnectwise.net`,
  `api-eu.myconnectwise.net`, `api-au.myconnectwise.net`, `api-staging.connectwisedev.com`.
- `clientId` — the integration's client GUID.

## Base URL

`https://{workspace}/v4_6_release/apis/3.0`

`v4_6_release` is the default codebase and is redirected to the partner's PSA
version server-side.

## Read objects

Object names are derived from the URL path and keep their service prefix:

`company/companies`, `company/contacts`, `company/configurations`,
`service/tickets`, `sales/opportunities`, `sales/activities`,
`project/projects`, `finance/agreements`, `procurement/products`,
`time/entries`, `schedule/entries`.

### Incremental sync

Reads filter on the standard `lastUpdated` audit field via the `conditions`
query parameter, e.g. `lastUpdated >= [2026-01-01T00:00:00Z] and lastUpdated < [...]`.
Both `Since` and `Until` are supported (datetimes are wrapped in square brackets
per ConnectWise condition syntax).

## Write objects

All read objects are writable. `POST /{object}` creates; `PUT /{object}/{id}`
replaces an existing record with the same flat shape that reads return. The
created/updated record is returned and its integer `id` is surfaced as the
record id.

## Pagination

Navigable pagination (RFC 5988). Requests use `pageSize=1000`; the response
`Link` header's `rel="next"` URL drives the next page and its absence ends paging.

## Notable quirks

- List responses are a **top-level JSON array** (no envelope).
- Rate limiting: HTTP 429 with a `Retry-After` header; integrations over
  ~1,000 req/min are throttled.
- Metadata is obtained by **response sampling** (one record per object) — no
  OpenAPI spec or discovery endpoint was available in the docs, so field types
  are not populated.
- `// TODO:` ConnectWise exposes ~3,100 endpoints; this first version covers the
  core persistent PSA entities. DELETE is documented on `/{id}` but the
  component Writer does not perform deletes — add `components.Deleter` to support it.
