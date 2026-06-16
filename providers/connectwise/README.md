# ConnectWise PSA Connector

## Auth

**Basic auth** — the Authorization header encodes `companyId+publicKey:privateKey` in base64.
Set the username to `{companyId}+{publicKey}` and the password to `{privateKey}`.

All requests also require a `clientId` header containing your integration's client GUID.

## Base URL

```
https://{region}.myconnectwise.net/v4_6_release/apis/3.0
```

The `region` metadata input defaults to `na`. Common values: `na`, `eu`, `au`, `aus`, `za`.

> **Note**: Cloud environments may use `api-{region}.myconnectwise.net`. If the `{region}.*` host
> returns errors, check your ConnectWise instance's actual host (see developer docs).

## Connector Metadata

| Field | Required | Description |
|---|---|---|
| `region` | Yes | Regional subdomain (default: `na`) |
| `clientId` | Yes | Integration clientId GUID |

## Read Objects

| Object | API Path | Incremental field |
|---|---|---|
| `service/tickets` | `/service/tickets` | `lastUpdated` |
| `companies` | `/company/companies` | `lastUpdated` |
| `contacts` | `/company/contacts` | `lastUpdated` |
| `projects` | `/project/projects` | `lastUpdated` |
| `activities` | `/sales/activities` | `lastUpdated` |
| `time/entries` | `/time/entries` | `lastUpdated` |
| `invoices` | `/finance/invoices` | `lastUpdated` |
| `schedule/entries` | `/schedule/entries` | `lastUpdated` |
| `opportunities` | `/sales/opportunities` | `lastUpdated` |

Incremental reads use the `conditions` query parameter:
`conditions=lastUpdated>[2016-08-20T18:04:26Z]`

## Pagination

Navigable pagination using `page` / `pageSize` query params.  
The API returns a `Link` response header with `rel="next"` and `rel="last"` URLs.  
The connector follows the `rel="next"` link until it is absent.  
Default page size: 1000 (maximum allowed).

## Rate Limits

~1,000 requests/minute (rolling window). Exceeding this returns HTTP 429 with a `Retry-After` header.

## Notable Quirks

- The `clientId` header is required on every request (separate from Basic auth credentials).
- Datetime conditions must be wrapped in square brackets: `lastUpdated>[2016-08-20T18:04:26Z]`.
- All list endpoints return a direct JSON array (no wrapper key).
- Write support is not implemented in this connector.
