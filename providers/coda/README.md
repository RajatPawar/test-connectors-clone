# Coda (Superhuman Docs) connector

Write-only deep connector for the Coda REST API v1 (Coda is rebranded as Superhuman Docs).

## Auth

API key, sent as `Authorization: Bearer <token>` (catalog entry `providers/coda.go`, unchanged).
A token acts with the permissions and workspace role of the user who created it; creating a doc
requires that user to be a Doc Maker (or Admin) in the target workspace. Tokens can also be
restricted to a single doc/table, which will make `POST /docs` fail with 403.

## Base URL

The catalog keeps `https://coda.io/apis`; the connector appends `v1`. The reference pages state
`https://coda.io/apis/v1`, while the OpenAPI `servers[]` now says
`https://docs.superhuman.com/apis/v1`. The catalog host was left as-is (existing entry).

## Write objects

| Object | Create | Update | Body |
|--------|--------|--------|------|
| `docs` | `POST /v1/docs` (201, returns the Doc) | `PATCH /v1/docs/{docId}` (200, returns `{}`) | RecordData sent as-is |

- Create accepts `title`, `folderId`, `timezone`, `sourceDoc` (doc id to copy), `initialPage`.
- Update accepts `title`, `iconName` only. Note the Doc read shape calls the title `name`.
- `RecordId` is the doc `id`. The create response also has a `requestId` for the async
  mutation (see `GET /mutationStatus/{requestId}`); it is not returned as the record id and the
  connector does not poll. The doc may take a few seconds to be fully usable.

## Metadata

Static `metadata/schemas.json` (Doc fields from the OpenAPI spec, plus the write-only inputs
`title`, `timezone`, `iconName`, `initialPage`).

## Not implemented (requested)

- `tables`, `columns`: the API has no create/update endpoint (GET only).
- `rows`: `POST /docs/{docId}/tables/{tableIdOrName}/rows` and
  `PUT .../rows/{rowIdOrName}` need a doc id and table id in the path (dynamic URL inputs), and
  insert is a bulk array endpoint. Use the proxy.

## Rate limits (docs)

Writes (POST/PUT/PATCH): 10 requests / 6 s per user; listing docs: 4 / 6 s. 429 on excess.
