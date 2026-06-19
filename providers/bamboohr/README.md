# BambooHR Connector

Read-only connector for the [BambooHR API](https://documentation.bamboohr.com/reference).

## Authentication

Basic authentication — API key as the username, blank password.
The API key is generated per-user in BambooHR under **Account → API Keys**.

## Connector Metadata

| Key | Description |
|-----|-------------|
| `workspace` | The BambooHR company domain (e.g. `mycompany` in `mycompany.bamboohr.com`) |

## Supported Objects

| Object | URL Path | Notes |
|--------|----------|-------|
| `api/v1/employees` | `/api/v1/employees` | Employee directory |
| `webhooks` | `/api/v1/webhooks` | Configured webhook subscriptions |
| `fields` | `/api/v1/meta/fields` | All available employee field definitions (root-array response) |
| `api/v1/meta/timezones` | `/api/v1/meta/timezones` | Supported timezone list |
| `custom-reports` | `/api/v1/custom-reports` | Saved custom report definitions |
| `api/v1_2/datasets` | `/api/v1_2/datasets` | Available dataset catalog (v1.2) |

## Incremental Read

BambooHR list endpoints do not expose a universal `updated_at`-style filter at
the collection level; `Since`/`Until` filtering is not applied by this connector.
Callers receive all records on each read. If time-scoped reads are needed per
object, a `// TODO` is left in `handlers.go`.

## Pagination

Page-number pagination via `page` and `pageSize` query parameters. Default page
size is 500. The connector advances the page when a full page is returned.

## Not Implemented

- Write operations (read-only connector)
- `api/v1/time_off/requests` — requires mandatory `start`/`end` query params; not a standard list endpoint
- `api/v1/time_tracking/timesheet_entries` — same mandatory date-range constraint
- `api/v1/meta/users` — response is keyed by userId, not a standard array
- Rate limits: not published by BambooHR; no entry in limiter defaults

## Running the Live Read Binary

```bash
# Print all objects to stdout
go run ./test/bamboohr/read

# Read a single object
go run ./test/bamboohr/read -object webhooks -fields id,name

# Capture all objects to /tmp/bamboohr-capture/
go run ./test/bamboohr/read -out /tmp/bamboohr-capture/
```

Credentials are loaded from the path specified by `BAMBOOHR_CREDS_PATH` (or the
default credential scanning path). The file must contain `apiKey` and `workspace`
fields as JSON.
