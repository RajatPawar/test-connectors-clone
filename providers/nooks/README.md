# Nooks Connector

Read-only connector for the [Nooks Sequencing API](https://www.nooks.in).

## Auth

Bearer token in the `Authorization` header (`Bearer <token>`). The API
accepts either a long-lived, workspace-scoped API key (prefixed
`nooks-api-`, generated under Developer Settings → API Keys) or a 1-hour
OAuth2 access token — both formats are validated on the same header, so this
connector is modeled as a plain `ApiKey` provider.

## Base URL

Single fixed production server: `https://partner-api.nooks.in/v1`. There is
no per-tenant subdomain or path segment — the workspace is resolved entirely
from the bearer credential — so no connector metadata fields are declared.

## Read objects

`calls`, `tasks`, `users`, `emails`, `accounts`, `mailboxes`, `prospects`,
`sequences`, `sequenceSteps`, `sequenceStates`, `callDispositions`.

`emailTemplate` is intentionally not exposed: the spec only defines
`GET /emailTemplate/{id}` (singular, no list endpoint) — it's reached via a
sequence step's `template` reference, not a company-wide collection.

### Incremental sync

Every object filters on `updatedAt`, but the query-parameter shape is not
uniform:

- `calls`, `users`, `emails`, `accounts`, `mailboxes`, `sequences`,
  `sequenceSteps`, `sequenceStates`: `filter[updatedAt][gte]` (inclusive) +
  `filter[updatedAt][lt]` (**exclusive** upper bound).
- `prospects`: defines its own `filter[updatedAt][gte]` +
  `filter[updatedAt][lte]` pair instead of referencing the shared component —
  its upper bound is **inclusive**, unlike every other object.
- `tasks`, `callDispositions`: the spec has **no** `filter[updatedAt]`
  parameter at all. These are filtered connector-side (fetch every page,
  discard records outside `Since`/`Until` locally against the record's
  `updatedAt` field) since the docs make no ordering guarantee for either
  object, so pagination cannot short-circuit early.

## Pagination

Cursor-based: `page[size]` (default 50, max 100) and an opaque
`page[after]`/`page[before]` cursor. Each response includes
`links.{next,prev,first}`; the spec documents `links.next` as a relative
reference that must be resolved against the request's base URL, even though
its own worked examples show a full absolute URL. The connector resolves
whichever form is returned via `url.ResolveReference` (no-op if already
absolute) and replays the resolved `links.next` URL verbatim as the next
request — it already encodes the original filters.

## Notable quirks

- Error envelope is standard HTTP status codes (400/401/403/404/429/500) with
  a `{"error": {"code", "message"}}` body — handled by `common.InterpretError`,
  no custom error handler needed.
- `accounts` only returns accounts sourced from a connected CRM (Salesforce/
  HubSpot); `callDispositions` only returns `nooks_sep`-type dispositions.
  Both can legitimately return an empty list in a workspace without a
  connected CRM/SEP — not a connector bug.
- Several fields (`owner`, `prospect`, `account`, `sequence`, etc.) are
  nested-only `{id, _href}` reference stubs; this connector does not expand
  them via `?include=`, so builders see only the stub unless a future round
  adds relationship hydration.

## Not implemented this round

Writes (tasks, sequences, sequenceStates, notes, prospect CRM sync) are
documented in the spec but out of scope — this connector is read-only per
this round's requirements.
