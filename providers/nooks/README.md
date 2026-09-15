# Nooks

Read-only connector for [Nooks](https://www.nooks.in)' partner API.

## Auth

`Authorization: Bearer <token>` — API key auth (`AuthType: ApiKey`, header
attachment, `Bearer ` value prefix). The header accepts either a long-lived
`nooks-api-...` key (Developer Settings -> API Keys, full workspace read) or a
short-lived OAuth2 access token issued by `oauth.nooks.in`; Nooks auto-detects
which format was supplied on the same header, so no separate OAuth wiring is
needed in this connector.

## Base URL

`https://partner-api.nooks.in/v1` — a single fixed production server (no
per-tenant subdomain or path variable; the workspace is resolved from the
bearer token itself), copied verbatim from the OpenAPI spec's `servers[0].url`.

## Read objects

All 11 objects with a real `GET` list endpoint in the spec, all incremental on
`updatedAt`: `calls`, `users`, `emails`, `accounts`, `mailboxes`, `sequences`,
`sequenceSteps`, `sequenceStates`, `prospects`, `tasks`, `callDispositions`.

- Most objects filter via `filter[updatedAt][gte]` + `filter[updatedAt][lt]`
  (exclusive upper bound).
- `prospects` defines its own `filter[updatedAt][gte]` + `[lte]` instead
  (**inclusive** upper bound) — it doesn't share the other objects' filter
  component.
- `tasks` and `callDispositions` have no `filter[updatedAt]` parameter at all,
  so every page is fetched and filtered **connector-side** against each
  record's `updatedAt`. Ordering isn't documented for either, so pagination
  always walks every page rather than stopping early.

`emailTemplate` is not exposed: the spec only has `GET /emailTemplate/{id}`
(no list endpoint), reached only via a sequence step's `template` reference.

## Write objects

None — this connector is read-only this round (`Write: false`). The API does
support writes for `tasks`, `sequences`, `sequenceStates`, and prospect/account
notes, but that's out of scope here.

## Pagination

Cursor-based. Every list response is `{"data": [...], "links": {"next",
"prev", "first"}}`. `links.next` is documented as possibly a *relative*
reference ("resolved against the base URL of the request"), even though the
spec's worked examples show absolute URLs — this connector resolves whichever
form comes back against the producing request's URL, so both are handled
correctly. Page size defaults to the provider's max (100 via `page[size]`).

## Rate limits

Published, per-workspace, per-endpoint, fixed one-minute window (see
`docs/openapi_spec.json`'s "Rate Limiting" section). List-read endpoints (all
objects this connector reads) are capped at 300 requests/minute per endpoint;
`429` responses include a `Retry-After` header.
`server/shared/limiter/defaults.go` does not exist in this checkout (no
`server/` directory here), so this could not be wired in directly this round —
flagging the limit here for whoever adds it in the monorepo that does have
that file.

## Quirks

- The `Authorization` header accepts two different token formats
  interchangeably; the connector doesn't need to know or care which one a
  caller supplies.
- Reference-type fields (`owner`, `prospect`, `account`, `sequence`, `creator`,
  etc.) come back as `{id, _href}` stubs rather than expanded objects — the
  spec's `?include=` parameter can expand up to 3 of these inline, but this
  connector doesn't use it. `docs/schemas.json` already types these fields as
  objects, so they're passed through as-is rather than flattened.
- Error responses are standard HTTP status codes with a `{"error": {"code",
  "message"}}` body — handled by the default `common.InterpretError`, no
  custom error handler needed.
