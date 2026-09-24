# Coda (Superhuman Docs) connector

Write-only connector for the Coda / Superhuman Docs REST API
(<https://docs.superhuman.com/apis/v1>). It lets a customer application insert,
upsert and update **rows** inside existing Coda tables.

## Auth

API-key (personal API token) sent as `Authorization: Bearer <token>`. Coda's REST
API supports only personal tokens; there is no OAuth 2.0 for direct API access.

## Base URL

`https://docs.superhuman.com/apis` — host taken from the OpenAPI spec (`servers` /
"API Endpoint", which documents `https://docs.superhuman.com/apis/v1`). Coda
rebranded to Superhuman Docs; this is the canonical host in the current spec. (The
provider docs page still shows some legacy `coda.io` URLs.)

Per repo convention the provider base URL carries **no** version suffix, so the
`v1` version segment is not part of `BaseURL`; the connector's request builder
prepends it to every path (see `apiVersion` in `handlers.go`). Effective request
paths are therefore `https://docs.superhuman.com/apis/v1/...`.

## Objects (write)

The write target is a table's **rows** collection. Because a write needs its doc
and table context and those vary per request, they travel in the object name,
which is the rows path:

```
docs/{docId}/tables/{tableIdOrName}/rows
```

- **Insert** — no `RecordId`: `POST .../rows`, body `{"rows":[{"cells":[...]}]}`.
- **Upsert** — as insert, but include a `keyColumns` key (a list of column
  IDs/names) in `RecordData`; it is lifted to the top-level `keyColumns`.
- **Update** — `RecordId` is the `rowIdOrName`: `PUT .../rows/{rowIdOrName}`, body
  `{"row":{"cells":[...]}}`.

`RecordData` is a flat `column -> value` map. Each entry becomes a Coda cell
`{"column": <id/name>, "value": <value>}`. Prefer column **IDs** over names —
names are fragile and can be renamed by doc users.

## Asynchronous writes

Row mutations are asynchronous: the API returns **HTTP 202** with a `requestId`
(the edit is queued, not yet applied). The connector returns success and puts the
full response — including `requestId` (and `addedRowIds` for inserts) — in
`WriteResult.Data`. To confirm completion, the caller polls
`GET /mutationStatus/{requestId}`. This reconciles with the customer's
synchronous expectation only after that poll succeeds (see the spec's open
question); a poll-until-complete step is not performed inside this connector.

## Rate limits

Writing data (POST/PUT/PATCH): 10 requests / 6 seconds. Writing doc **content**:
5 requests / 10 seconds. Responses use HTTP 429 when exceeded.

## Columns — read-only in the Coda API

The acceptance spec mentions writing "rows and columns", but the Coda
(Superhuman Docs) API has **no column-write endpoint**. The only column
operations are `listColumns` (`GET /docs/{docId}/tables/{tableId}/columns`) and
`getColumn` (`GET .../columns/{columnId}`) — both read-only. Columns (the table
schema) are created and edited only through the Coda UI. Per-column *values* are
written as part of a row: each entry in `RecordData` becomes a cell targeting an
existing column. So this connector satisfies the "column" part of the spec by
setting column values on row writes; it cannot create or alter columns, because
the provider offers no API for it.

## Not implemented

- Read / subscribe (out of scope for this connector).
- Column create/update — **not possible**: the Coda API has no column-write
  endpoint (see above).
- Row delete (`DELETE .../rows` and `.../rows/{rowId}`) — the API supports it, but
  it was not requested. Add via `components.Deleter` if needed.
