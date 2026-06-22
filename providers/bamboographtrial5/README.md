# BambooHR Connector

Read-only connector for the [BambooHR API](https://documentation.bamboohr.com/reference).

## Auth

Basic authentication. Use your BambooHR API key as the username; the password field is ignored by the API (any value is accepted, typically "x"). Generate an API key in **Account → API Keys** within the BambooHR web UI.

## Connector Metadata

| Field | Description |
|---|---|
| `companyDomain` | Your BambooHR subdomain (the `{company}` part of `{company}.bamboohr.com`). |

## Supported Objects

| Object | API Path | Incremental | Pagination |
|---|---|---|---|
| `jobs` | `/api/v1/applicant_tracking/jobs` | No | None |
| `api/v1/hris/org/locations` | `/api/v1/hris/org/locations` | `createdAt` only (TODO: no updatedAt filter) | page/pageSize |
| `schedules` | `/api/v1/scheduling/schedules` | `updatedAt` ✓ | page/pageSize |
| `policies` | `/api/v1/meta/time_off/policies` | No | None |
| `benefitcoverages` | `/api/v1/benefitcoverages` | No | None |
| `whos_out` | `/api/v1/time_off/whos_out` | `start` date window | None |

## Notes

- `api/v1/hris/org/locations` uses 0-indexed page numbers; `schedules` uses 1-indexed.
- `whos_out` returns a date-sorted list of employees out and holidays. Pass `Since` to scope the start date.
- The `policies` and `benefitcoverages` endpoints are simple list endpoints with no documented pagination.
- `jobs` returns all non-deleted job openings by default.
