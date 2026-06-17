# Tipalti Connector

Read-only connector for the [Tipalti API](https://documentation.tipalti.com/reference/introduction).

## Authentication

OAuth 2.0 Authorization Code flow. `ExplicitScopesRequired: true` — callers must request the appropriate scopes when initiating the OAuth flow.

Token endpoint: `https://sso.tipalti.com/connect/token`

## Supported Objects

| Object | Incremental (Since/Until) | Notes |
|---|---|---|
| `custom-fields` | No | No time filter documented |
| `gl-accounts` | No | No time filter documented |
| `invoices` | No | No time filter documented |
| `payees` | Yes | `lastChangeDateTimeUTC >= / <=` filter |
| `payer-entities` | No | No time filter documented |
| `payment-terms` | No | No time filter documented |
| `payments` | No | No time filter documented |
| `tax-codes` | No | No time filter documented |

Only `payees` exposes a server-side time filter (`lastChangeDateTimeUTC`). All other objects are read in full each sync.

## Pagination

Cursor-based: `pageSize=200` (API max) + `pageCursor` query param. The response envelope contains `pageInfo.hasNextPage` and `pageInfo.nextPageCursor`. Records are in the `items` array.

## Schema

Fields are sampled at runtime from the first record of each object (`pageSize=1` request). No static schema is included because Tipalti does not provide an OpenAPI spec.

## Rate Limits

1500 requests per minute (rolling window). See [Tipalti rate limiting docs](https://documentation.tipalti.com/reference/rate-limiting).

## Base URL

Production: `https://api-p.tipalti.com/api/v1`
