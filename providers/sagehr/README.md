# Sage HR Connector

Read-only connector for the [Sage HR API](https://developer.sage.com/hr/docs/v1.0.0/guides/get-started/quick-start).

## Auth

API key, sent in the `X-Auth-Token` header. Generated per-user under
**Settings > Integrations > API** (requires an Admin user); invalidated if
that admin loses admin rights — expect silent 401s in that case.

## Base URL

Per-tenant subdomain: `https://{workspace}.sage.hr/api`. `workspace` is the
subdomain (e.g. `mycompany` in `https://mycompany.sage.hr`).

## Read objects

`teams`, `employees`, `positions`, `termination-reasons`,
`terminated-employees`, `recruitment/positions`, `leave-management/policies`,
`leave-management/requests`, `documents/categories`,
`employees/compensations`, `employees/custom-fields`,
`employees/leave-management/balances`, `recruitment/positions/applicants`.

`employees` always requests the optional `team_history`,
`employment_status_history`, `position_history` collections (boolean-gated,
free to include).

**Incremental sync**: no object exposes an `updated_at`-style cursor. Only
`leave-management/requests` supports real time filtering (`from`/`to`, date
range under 65 days, defaults to current month if omitted). The connector
always chunks the range into 60-day windows and advances through them via
`NextPage` until `Until` (or now) — including on a full sync (`Since` zero),
where it substitutes a fixed `defaultLeaveRequestLookbackYears` (5 years,
see `parse.go`) instead of leaving `from`/`to` unset. Leaving them unset would
silently limit a full sync to the API's own current-month-only default. There
is no documented way to discover the true earliest leave request, so this
lookback is an explicit assumption — flagged for human review, not derived
from the docs. Every other object is a full read each call — documented here
rather than fabricated.

## Pagination

`meta.next_page` (int or null) drives continuation, but the query param
convention differs:

- `?page=N` — core endpoints, fixed page size of 50 (no size param accepted).
- `?page=N&per_page=M` (default 30, max 100) — `recruitment/positions` and
  `recruitment/positions/applicants`.
- No `meta` at all, full `data` array in one response —
  `leave-management/policies`, `documents/categories`,
  `employees/custom-fields`, `employees/leave-management/balances`.

## Fan-out objects

Sage HR has no company-wide listing for compensations, custom fields, leave
balances, or recruitment applicants — only `/employees/{id}/...` or
`/recruitment/positions/{id}/...`. The connector lists the parent object,
then fans out one request per parent id (bounded concurrency; see
`handlers.go`), flattening results; overall pagination still advances through
the parent list. `employees/compensations` and
`employees/leave-management/balances` records have no natural `id` field in
the provider's response, so `ReadResultRow.Id` is left empty for those two.

## Metadata (schema)

No schema/describe endpoint and no `schemas.json`. Fields are sampled live
from one record per object. Fan-out objects need one extra internal call to
fetch a real parent id to sample from.

## Not implemented this round (see `supports.go`)

- `documents`: only `POST /documents` exists (multipart upload), no GET.
- `leave-management/kit-days`: requires both `policy_id` and `employee_id`
  with no way to enumerate valid pairs.
- Applicant actions (a third fan-out level: positions → applicants → actions).
- `onboarding/*`, `offboarding/*`, `performance/goals/*`,
  `leave-management/reports/individual-allowances`, `vikarina/*` (a
  third-party payroll export surface) — out of scope for this round.

## Rate limits

Not published in the docs. `maxConcurrentChildFetch = 4` (handlers.go) is a
conservative default, not derived from a documented limit.
