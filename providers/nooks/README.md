# Nooks

Read-only connector for the [Nooks Sequencing API](https://partner-api.nooks.in/v1).

## Auth

Bearer token in the `Authorization` header (`Authorization: Bearer <token>`).
Nooks accepts two token formats on the same header — a long-lived workspace
API key (`nooks-api-...`) or a short-lived OAuth 2.0 access token — the API
validates both. Modeled here as `AuthType: ApiKey` with a `Header` attachment
and `ValuePrefix: "Bearer "` (see `providers/nooks.go`), since this codebase
has no separate "Bearer" auth type.

## Base URL

`https://partner-api.nooks.in/v1` — copied verbatim from the OpenAPI spec's
single `servers` entry. No templated/region segments; the spec declares only
one production server.

## Read objects

All objects below come straight from `metadata/schemas.json` (derived from
the OpenAPI spec) and use cursor pagination via `page[size]`.

| Object | Incremental field | How |
|---|---|---|
| `calls` | `updatedAt` | `filter[updatedAt][gte]` / `filter[updatedAt][lt]` |
| `users` | `updatedAt` | same as above |
| `emails` | `updatedAt` | same as above |
| `accounts` | `updatedAt` | same as above |
| `mailboxes` | `updatedAt` | same as above |
| `sequences` | `updatedAt` | same as above |
| `sequenceSteps` | `updatedAt` | same as above |
| `sequenceStates` | `updatedAt` | same as above |
| `prospects` | `updatedAt` | `filter[updatedAt][gte]` / `filter[updatedAt][lte]` (note: `lte`, not `lt` — this object's own inline params differ from the shared `FilterUpdatedAtGte`/`FilterUpdatedAtLt` parameters used everywhere else) |
| `tasks` | `updatedAt` | **no server-side filter exists** — the spec only offers `filter[dueAt]`. Since/Until is applied connector-side (fetch every page, discard out-of-window records) |
| `callDispositions` | `updatedAt` | **no server-side filter exists at all**. Same connector-side filtering as `tasks` |

Not implemented (see `supports.go` for full reasoning):
- `emailTemplate` — read-by-ID only, no list endpoint, so not a pollable object.
- `notes` — create-only (`POST .../notes`), no GET/list endpoint.
- Action endpoints (`/tasks/{id}/skip`, `/tasks/{id}/complete`,
  `/sequenceStates/{id}/actions/finish`, `/integrations/prospects/sync`) — verbs,
  not objects.

## Write objects

None. This connector is read-only this round (`Write: false` in `ProviderInfo`).
The API does support write endpoints for tasks, sequences, sequenceStates, notes,
and CRM prospect sync — out of scope here.

## Pagination

Cursor-based. Every list response is `{"data": [...], "links": {"next", "prev", "first"}}`.
`links.next` is a ready-to-use URL (already carrying `page[size]`/`page[after]`);
the connector passes it straight through as `NextPage` and reuses it verbatim on
the next request. Default page size 50, max 100 (`page[size]`).

Note: the spec's `PaginationLinks` schema description calls these links
"relative references", but every example in the same spec shows full absolute
URLs. We follow the examples (per project convention: examples win over
inconsistent prose) — see the TODO comment in `handlers.go` if a live capture
ever shows a genuinely relative link.

## Notable quirks

- All list endpoints share one envelope shape and response key (`data`), so a
  single generic `records()`/`nextRecordsURL()` pair in `parse.go` covers every
  object — no per-object special-casing needed.
- `prospects` uses `lte` (inclusive) for its upper `updatedAt` bound while every
  other object uses `lt` (exclusive) — copied exactly as documented, not
  normalized.
- Rate limits are published (per-workspace, per-endpoint, one-minute window;
  300 req/min for list GETs, 600 req/min for by-ID GETs) — see
  `server/shared/limiter/defaults.go` TODO / openapi_spec.json `info.description`.
