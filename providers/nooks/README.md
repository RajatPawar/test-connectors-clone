# Nooks

Read-only connector for the [Nooks Sequencing API](https://partner-api.nooks.in/v1).

## Auth

Bearer token in the `Authorization` header (`Authorization: Bearer <token>`).
Nooks auto-detects two token forms on the same header:

- a long-lived, workspace-scoped API key prefixed `nooks-api-` (Developer
  Settings → API Keys), or
- a 1-hour OAuth2 access token issued via authorization-code + PKCE from
  `https://oauth.nooks.in`.

`GET /me` validates either form. The connector is wired as a standard `ApiKey`
provider (header attachment, `Bearer ` prefix) — either token type can be
supplied as the "API key" value.

## Base URL

`https://partner-api.nooks.in` — the OpenAPI spec declares a single fixed
production server (`https://partner-api.nooks.in/v1`) with no per-tenant
subdomain or path segment. Per repo convention, `ProviderInfo.BaseURL` omits
the `/v1` version segment; `buildReadRequest` prepends it when building each
request path, so the actual request URL is unchanged. The workspace is
determined entirely by the credential, so no connector metadata field is
required.

## Read objects

All 11 top-level `GET` list endpoints in the OpenAPI spec: `calls`, `tasks`,
`users`, `emails`, `accounts`, `mailboxes`, `prospects`, `sequences`,
`sequenceSteps`, `sequenceStates`, `callDispositions`.

Not included: `emailTemplate` — only exposed as `GET /emailTemplate/{id}`
(singular, no list endpoint), so it isn't a valid top-level read object.

Incremental sync uses `updatedAt`:

- Most objects (`calls`, `users`, `emails`, `accounts`, `mailboxes`,
  `sequences`, `sequenceSteps`, `sequenceStates`) send the server-side
  `filter[updatedAt][gte]`/`filter[updatedAt][lt]` query params from
  `Since`/`Until`.
- `prospects` is the one exception with an inclusive upper bound: it uses
  `filter[updatedAt][lte]` instead of `[lt]` (confirmed in the OpenAPI spec —
  it defines its own inline parameter rather than referencing the shared
  `FilterUpdatedAtLt` component).
- `tasks` and `callDispositions` have **no** `filter[updatedAt]` parameter at
  all. For these, every page is fetched and filtered connector-side against
  each record's `updatedAt` field (see `handlers.go` /
  `objectsWithoutUpdatedAtFilter`). Record ordering within a page isn't
  documented, so pagination is never short-circuited early for these two
  objects — correctness over efficiency.

## Pagination

Cursor-based: `page[size]` (default 50, max 100 — this connector clamps any
larger requested size down to 100) and `page[after]`. Every list response is
wrapped as `{"data": [...], "links": {"next": ..., "prev": ..., "first": ...}}`;
`links.next` is `null` on the last page. The spec documents `links.next` as a
relative reference that must be resolved against the request's base URL, but
at least one response example in the spec shows a full absolute URL instead —
the connector resolves either case via `url.ResolveReference`. The resolved,
already-absolute next-page URL is re-issued as-is (it carries forward the
original query params), so no filter/page-size re-application is needed on
subsequent pages.

## Notable quirks

- Relationship fields (`owner`, `prospect`, `sequence`, `sequenceStep`,
  `callDisposition`, `account`, `creator`, `template`, etc.) are nested
  `{id, _href}` reference objects, not flat foreign keys, and are left nested
  in `Raw` — the API's optional `include` param (max 3, GET-only) that would
  hydrate them inline is not used by this connector.
- The `/calls` list/read schema does **not** include a transcript or rep
  notes — those (`transcriptUrl`, `notes`) exist only in the `call.finalized`
  webhook payload, which is out of scope for a polled read connector.
- Standard HTTP error codes with a `{"error": {"code", "message"}}` body
  (verified live: a 401 for an invalid token returns exactly this shape), so
  `common.InterpretError` is used as-is with no custom error handler.
- Rate limits (documented, not yet added to
  `server/shared/limiter/defaults.go` — see TODO below): list reads are
  300 requests/minute per endpoint; read-by-id is 600/minute per endpoint.
  Responses carry `X-RateLimit-*` headers; 429s carry `Retry-After`.

## TODOs / open gaps

- Rate limits are documented per-endpoint but not yet wired into
  `server/shared/limiter/defaults.go` (out of scope for this read-only
  connector round; see docs URL above for the limits table).
- This round is deliberately read-only: `tasks`, `sequences`,
  `sequenceStates`, notes, and CRM prospect sync all have write/action
  endpoints in the spec (`createTask`, `updateTask`, `createSequence`,
  `createSequenceState`, `/prospects/{id}/notes`, `/integrations/prospects/sync`,
  etc.) that are intentionally not implemented here.
- **Auth model open question (flagged for human review):** the customer's
  stated preference is per-user OAuth (each end user connects with their own
  Nooks login). The OpenAPI spec's `securitySchemes` only documents a single
  Bearer-token scheme that accepts either a workspace API key or an OAuth2
  access token on the same header — it does not describe a per-user OAuth
  authorization-code flow independent of the workspace-scoped key. This
  connector implements the documented Bearer/ApiKey scheme (either token type
  fits through the same field); reconciling it with the desired per-user OAuth
  UX is a product/access-model decision, not something resolvable from the
  docs alone.
- Live capture with the creds available this round returned
  `401 INVALID_TOKEN` from the real API — the configured API key value is not
  a valid Nooks credential. This is an account/credential problem, not a
  connector bug (the request reached `https://partner-api.nooks.in/v1/calls`
  and got a well-formed, correctly-parsed error back); a human needs to supply
  a valid `nooks-api-...` key or OAuth2 token before live capture can proceed.
