# Lightfield connector

Read-only deep connector for the Lightfield CRM REST API (public beta).

## Auth and requests

- API key, sent as `Authorization: Bearer sk_lf_...`. Keys are created by workspace admins with
  scopes; reads need `accounts:read`, `contacts:read`, `opportunities:read` and `members:read`.
  The docs describe no OAuth flow for the REST API (OAuth 2.1 covers only Lightfield's MCP server).
- Base URL `https://api.lightfield.app`; object paths are under `/v1`.
- Every request carries `Lightfield-Version: 2026-03-01` (the only version the spec lists).
  Requests without it fail with error code `version_header`. The catalog cannot attach a fixed
  header, so proxy callers would have to send it themselves.

## Objects

| Object          | Endpoint                 | Metadata source                                   |
|-----------------|--------------------------|---------------------------------------------------|
| `accounts`      | `GET /v1/accounts`       | catalog + `GET /v1/accounts/definitions`          |
| `contacts`      | `GET /v1/contacts`       | catalog + `GET /v1/contacts/definitions`          |
| `opportunities` | `GET /v1/opportunities`  | catalog + `GET /v1/opportunities/definitions`     |
| `members`       | `GET /v1/members`        | catalog only (no definitions endpoint)            |

The customer's "users" are workspace members (`members`). Note that `$owner` relationships point at
user IDs, which are linked to members through each member's `$user` relationship.

The catalog (`internal/metadata/objectsMetadata.json`) is generated from the OpenAPI spec by
`go run ./scripts/openapi/lightfield/metadata`. It provides the top-level record properties and the
fixed member fields. Definitions add every standard and custom field and relationship, with labels,
select options, `readOnly` and custom-ness (custom keys have no `$` prefix).

## Record shape

Records are `{id, createdAt, updatedAt, httpLink, externalId, fields, relationships}` where each
field is `{value, valueType}` and each relationship is `{cardinality, objectType, values}`.
Read flattens them: `fields.<key>.value` becomes `<key>` and `relationships.<key>.values` (related
IDs) becomes `<key>`, keeping the API's keys (`$name`, `icp-score`). Select fields hold option IDs.
`Raw` is untouched.

## Pagination

`limit` (max 25) and `offset`. The next offset is `offset + page size`; reading stops on a page
shorter than `limit` or when the offset reaches `totalCount`.

## Incremental read

List endpoints have no updated-since filter and return records newest-created first, so `Since`
and `Until` are applied connector-side to `updatedAt` (falling back to `createdAt` when `updatedAt`
is null), after reading every page. DATETIME field filters exist, but the docs do not say whether
`updatedAt` can be filtered that way; this needs confirming with Lightfield.

Lists are served from a search index and may lag recent changes.

## Rate limits

25 requests per second per organization for each of the Search (list) and Read (definitions)
categories. https://docs.lightfield.app/using-the-api/rate-limits
