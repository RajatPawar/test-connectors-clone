# Ashby (self-test) connector

A read-only, docs-only rebuild of the Ashby integration, built independently
from `./docs/` for validation purposes. See "Relationship to `providers/ashby`"
below for why this exists alongside the pre-existing `providers/ashby` package.

## Auth

HTTP Basic — the Ashby API key is sent as the Basic-auth **username** with a
**blank password** (`Authorization: Basic base64("<API_KEY>:")`). Not a
Bearer token, not an `X-Api-Key` header. Missing key → 401; wrong/deactivated
key → 403; a valid key missing a module's permission → 403 with
`missing_endpoint_permission`.

## Base URL

Fixed: `https://api.ashbyhq.com`. No per-tenant subdomain, region, or
workspace variable — every endpoint is `https://api.ashbyhq.com/{category}.{method}`.

## Read objects

All eight are RPC-style **POST** `.list` calls (Ashby uses POST even for
reads — see "Notable quirks"):

| Object | Endpoint | Notes |
|---|---|---|
| `candidate` | `candidate.list` | |
| `application` | `application.list` | |
| `job` | `job.list` | |
| `jobPosting` | `jobPosting.list` | no cursor pagination; requests `includeUnpublishedJobPostings: true` |
| `offer` | `offer.list` | requires `offersRead` key permission |
| `opening` | `opening.list` | |
| `user` | `user.list` | requires `organizationRead` key permission |
| `interviewSchedule` | `interviewSchedule.list` | requires `interviewsRead` key permission |

**Incremental sync:** Ashby has no `updated_at`/`updated_since` query filter
on any list endpoint — only an opaque `syncToken` (requires persisting
provider state across syncs, which `ReadParams` doesn't carry) and, on some
objects, a `createdAfter` (creation-time) bound that would miss updates to
existing records. Per this repo's incremental-read rule ("filter by
`updated_at`, never `created_at`"), `Since`/`Until` are therefore applied
**client-side** in `parse.go` against each record's own update timestamp,
rather than sent as request params:
- Most objects: the record's top-level `updatedAt`.
- `offer` and `opening`: neither object exposes an `updatedAt` in its `.list`
  response at all. Every edit creates a new `latestVersion`, so
  `latestVersion.createdAt` is used as the best available proxy for "last
  modified". This is a documented gap, not a guess — see `connie_notes.md`.

A record with a missing or unparsable timestamp is kept rather than dropped,
since the docs are inconsistent about which timestamp fields are `required`
per object.

## Pagination

Cursor-based, in the JSON body: `cursor` (opaque, from the previous
response's `nextCursor`) and `limit` (default/max 100). The response envelope
is `{ success, moreDataAvailable, nextCursor, syncToken, results: [...] }` —
loop while `moreDataAvailable` is true. `jobPosting.list` is the one
exception: it has no cursor/limit params at all and always returns every
posting in a single response.

## Notable quirks

- **Every "read" is a POST.** Ashby's API is RPC-style
  (`/{category}.{method}`); list/info endpoints take `POST` with a JSON body,
  not `GET` with query params.
- **Records live under `results`**, not a bare top-level array.
- Confidential jobs/projects and non-offer private candidate fields are
  hidden from API keys by default; enabling them requires extra permissions
  in the Ashby web app, not anything this connector controls.
- Metadata source: no `openapi_spec.json`/`schemas.json` was provided in
  `./docs/` for this build, and there's no field-discovery endpoint for
  arbitrary objects. `metadata/schemas.json` was hand-authored directly from
  the field lists documented on each `.list` reference page (Priority 3-style
  response sampling, done from docs instead of a live call).
- Rate limits are not published in the scraped docs for the general RPC API
  (only `report.*`, which isn't in scope here, documents a limit: 15 req/min +
  3 concurrent per org). However, a live (unauthenticated) request to
  `candidate.list` returned `x-ratelimit-limit: 1000`,
  `x-ratelimit-remaining: 998`, and an `x-ratelimit-reset` ~60s out — so the
  general API appears to enforce roughly 1000 requests/minute per key,
  observed from real response headers rather than documented anywhere. See
  `connie_notes.md` for the raw evidence. `server/shared/limiter/defaults.go`
  does not exist in this repo checkout, so no entry was added there.
- **Live capture is currently blocked by test-fixture creds, not a connector
  bug.** `/home/appuser/forge/creds/ashby-creds.json` is regenerated between
  rounds with an `{"apiKey": "..."}` shape, but this connector's (docs-
  verified) `AuthType: Basic` requires `username`/`password` JSON keys —
  `test/capturekit` fails before making any HTTP call with
  `key not found: username,password`. Reshaping the file to
  `{"username": "<key>", "password": ""}` lets the request reach
  `https://api.ashbyhq.com/candidate.list` for real, but the fixture's key is
  a dummy value, so the live API correctly returns `401 Unauthorized`
  (confirmed with a matching plain `curl`) — a creds/account issue for a
  human to resolve, per CLAUDE.md's guidance on judging live-read errors, not
  something this package can fix.

## Relationship to `providers/ashby`

This repository already has a separate, pre-existing `providers/ashby`
package (read+write, OpenAPI-derived static schema). `providers/ashbyselftest`
was built independently from `./docs/` only, without referring to that
package's implementation choices. Both share the same provider identity:
`providers.Ashby` (`Provider` value `"ashby"`) — Go doesn't allow two catalog
entries for the same `Provider` constant, so `ashbyselftest` reuses it rather
than declaring a new one.

Since this run is read-only and `ashbyselftest` is the graded deliverable,
the shared catalog entry and the runtime wiring were both updated to match:
- **`providers/ashby.go`** — `Support.Write` was flipped from `true` to
  `false`, since a read-only connector cannot advertise write support at the
  catalog level (this level is shared, not per-package).
- **`connector/new.go`** — the `providers.Ashby` case now constructs
  `ashbyselftest.NewConnector` instead of the original `providers/ashby`
  package, so this package is the one actually instantiated at runtime.

The original `providers/ashby` package's source is untouched and still
compiles/tests on its own; it is simply no longer reachable from
`connector/new.go`.
