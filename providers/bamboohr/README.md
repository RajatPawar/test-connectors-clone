# BambooHR Connector

Read-only connector for the [BambooHR API](https://documentation.bamboohr.com/reference).

## Auth

HTTP Basic auth. The API key is used as the username; the password is arbitrary/blank
(BambooHR ignores it). Confirmed against the docs (e.g. "Create Employee" lists
`Credentials: Basic — username : password`) and the OpenAPI `securitySchemes.basic` entry
present on every object this connector reads.

## Base URL

`https://{{.workspace}}.bamboohr.com` — `workspace` is the company's BambooHR subdomain
(`companyDomain` in the docs/spec), collected as connector metadata.

## Read objects

| Object | Endpoint | Incremental field |
|---|---|---|
| `employees` | `GET /api/v1/employees` | Two-step: `GET /api/v1/employees/changed?since=` resolves changed IDs, then `filter[ids]` is applied to the main call. List Employees itself has no last-changed filter. |
| `employees/directory` | `GET /api/v1/employees/directory` | Not supported — endpoint has no time filter (documented gap). |
| `custom-reports` | `GET /api/v1/custom-reports` | Not supported — only exposes report `id`/`name`, no timestamp (documented gap). |
| `time_off/requests` | `GET /api/v1/time_off/requests` | `start`/`end` are *required* by the API; `Since`/`Until` map to them, defaulting to a 1-year-back/1-year-forward window when omitted. |
| Standard employee tables (`jobInfo`, `compensation`, `employmentStatus`, `contacts`, `emergencyContacts`, `dependents`, `earnings`, `bonus`, `commission`, `benefit_class`, `employeeVisas`, `employeeEducation`, `employeePassports`, `employeeDriverLicenses`, `employeeCertifications`, `employeeStockOptions`, `employeeAssets`, `employeeCreditCards`, `employeeCovidTests`, `employeeCovidVaccinations`, `employeeCovidVaccinationExemptions`, `employeeCovidExposures`, `employeeEquityGrants`, `levelsAndBands`, `employeeProjectPayRates`) | Full: `GET /api/v1/employees/all/tables/{table}`. Incremental: `GET /api/v1/employees/changed/tables/{table}?since=` | `since` (RFC3339) on the changed-data endpoint. |

Custom (company-defined) tables (e.g. `custom1`) are **not** registered — their names are
only discoverable per-company via `GET /api/v1/meta/tables`, a discovery endpoint, so they
can't be listed statically. `GET /api/v1/meta/fields` and `GET /api/v1/meta/tables` are
likewise not exposed as readable objects — they are field/table *catalogs*, not persistent
data records (see CLAUDE.md's "no synthetic objects" rule).

## Pagination

* `employees`: cursor-based, via `page[limit]` / `page[after]`. The response's
  `_links.next.href` is a ready-to-use absolute URL and is passed straight through as
  `ReadResult.NextPage`.
* `custom-reports`: offset-based, via `page` / `page_size`. `pagination.next_page` is
  likewise a ready-to-use absolute URL.
* `employees/directory`, `time_off/requests`, and every employee-table object: single page,
  no pagination — the API returns the full result set in one response.

## Notable quirks

* The "Get Changed Employee Table Data" response for employee tables groups rows by employee
  ID (`{"employees": {"<id>": {"rows": [...], "lastChanged": ...}}}`), unlike every other
  shape in this connector. `changedTableRecords` in `parse.go` flattens it back into individual
  rows, attaching `employeeId` and `lastChanged` to each one so callers see a consistent shape
  regardless of which endpoint served the data.
* `List Employees` records are keyed by `employeeId`, not `id`; every other object here uses
  `id`. See `idFieldForObject` in `parse.go`.
* Metadata is a static, OpenAPI-derived `schemas.json` (Priority 1 per CLAUDE.md), hand-trimmed
  down to the objects this connector actually exposes and renamed to match the object names
  above (the raw scraped `schemas.json` uses raw URL-segment keys like `api/v1/employees` and
  is missing `employees/directory` entirely).
* Write is intentionally out of scope for this connector (`Write: false`); see `connie_notes.md`
  for what a future write implementation would need to cover.
