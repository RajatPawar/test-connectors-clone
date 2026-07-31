# Nooks

Read-only connector for the Nooks Sequencing API (Sales Engagement Platform).

## Auth

Bearer token in the `Authorization` header: `Authorization: Bearer <token>`.
The API accepts either a long-lived, workspace-scoped API key (prefixed
`nooks-api-`, from Developer Settings -> API Keys, full read access, no
scopes) or a short-lived OAuth 2.0 access token, auto-detected on the same
header. Modeled here as `ApiKey` with `AttachmentType: Header` and
`ValuePrefix: "Bearer "` -- the API key is the simpler, longer-lived credential
for a server-to-server sync.

## Base URL

`https://partner-api.nooks.in/v1` -- a single fixed production server. There
is no per-tenant subdomain or path variable; the workspace is resolved from
the token, so no connector metadata fields are required.

## Read objects

- `prospects`, `accounts`, `users`, `sequences`, `sequenceStates`, `tasks`,
  `emails`, `calls`

All incrementally sync on `updatedAt`, using `filter[updatedAt][gte]`
(Since) and `filter[updatedAt][lt]` (Until, exclusive), **except**:

- **prospects**: uses `filter[updatedAt][gte]` / `filter[updatedAt][lte]`
  (the upper bound is *inclusive* here, unlike every other object).
- **tasks**: has no `filter[updatedAt]` parameter at all (only
  `dueAt`/`status`/`action`/`priority`/`completed`/`sequenceState[state]`).
  Since/Until are applied connector-side in `parseReadResponse`: every page
  is fetched and records outside the window are discarded locally.
- **sequences**: `updatedAt` is a *derived* value (max of the sequence's own
  update time and any of its steps' update times), so incremental syncs here
  also pick up step-level edits.

Not yet implemented (documented, generically supportable, out of scope for
this run): `mailboxes`, `sequenceSteps`, `callDispositions`. `emailTemplate`
has no list endpoint at all (`GET /emailTemplate/{id}` only, reached via a
sequence step's `template` reference) so it isn't a listable object.

## Pagination

Cursor-based via `page[size]` (default 50, max 100) and `page[after]`. Each
list response is `{"data": [...], "links": {"next": ..., "prev": ..., "first": ...}}`.
`links.next` may be a *relative* reference (path + query only); this
connector resolves it against the URL of the request that produced it before
storing it as the next-page token, so subsequent reads can `GET` it directly.
`links.next` is `null` on the last page.

## Notable quirks

- Relationship fields (`owner`, `prospect`, `account`, etc.) are nested
  `{id, _href}` reference objects, not flat foreign keys. This connector does
  not hydrate them via `?include=` -- they are returned as-is in `Raw`.
- `/accounts` only returns CRM-sourced (Salesforce/HubSpot) accounts, and can
  legitimately be empty in a workspace with no connected CRM.
- `/users` only returns users with an active seat assignment.
- Standard error envelope: `{"error": {"code": ..., "message": ...}}`, but
  HTTP status codes are standard, so this connector uses
  `common.InterpretError` directly (no custom `errors.go`).
- Read-only in this run: no `Writer`, `Write: false` in `ProviderInfo.Support`.
