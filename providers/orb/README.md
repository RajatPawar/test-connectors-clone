# Orb connector

Read-only deep connector for Orb's billing REST API.

## Auth and requests

- An Orb API key (Orb dashboard, Developers > API keys), sent as `Authorization: Bearer <key>`.
- Base URL `https://api.withorb.com`; every path is under `/v1`.
- Errors follow RFC 7807 (`type`, `status`, `title`, `detail`) with standard HTTP status codes,
  so `common.InterpretError` handles them
  ([Error responses](https://docs.withorb.com/api-reference/error-responses)).

## Objects

| Object          | Endpoint                | Notes                                             |
|-----------------|-------------------------|---------------------------------------------------|
| `customers`     | `GET /v1/customers`     |                                                   |
| `subscriptions` | `GET /v1/subscriptions` | metadata partly sampled, see below                |
| `invoices`      | `GET /v1/invoices`      | sends `status[]` for every status                 |
| `plans`         | `GET /v1/plans`         |                                                   |
| `items`         | `GET /v1/items`         |                                                   |
| `credit_notes`  | `GET /v1/credit_notes`  |                                                   |
| `coupons`       | `GET /v1/coupons`       | sends `show_archived=true`                        |

List invoices returns only `issued`, `paid` and `synced` invoices unless filtered by status, and
list coupons only active ones, so both ask for every record explicitly.
Fields are the record's top-level keys as Orb returns them; nested objects (`customer`, `plan`,
`line_items`...) are kept as values, not flattened.

## Metadata

Orb has no describe endpoint and our docs carry no OpenAPI spec, so
`internal/metadata/objectsMetadata.json` is written by hand from the response example of each
list endpoint in the API reference (enum values for `invoices.status`, `plans.status` and
`customers.payment_provider` come from the reference's enum lists).
The reference has no response schema for subscriptions: their catalog entry lists the fields a
live response confirmed, and `ListObjectMetadata` samples one subscription
(`GET /v1/subscriptions?limit=1`) to add any other top-level keys with an inferred type.

## Pagination

Cursor based: `limit` (default 20, max 100; the connector asks for 100) and `cursor`. While
`pagination_metadata.has_more` is true, `next_cursor` goes back as `cursor` on the same request
([Pagination](https://docs.withorb.com/api-reference/pagination)).

## Incremental read

None: every read is a full read and `Since`/`Until` are ignored. No list endpoint filters on an
update time and records carry no `updated_at`. Customers, plans and credit notes filter on
`created_at`, and invoices on `invoice_date`, but filtering on those would never deliver a
record again after it changes (an invoice being paid or voided, a customer's email changing).

## Rate limits

Read-only (`GET`) requests: 100/s in live mode, 50/s in test mode, 10/s on trial accounts; a 429
should be retried with backoff ([Rate limits](https://docs.withorb.com/api-reference/rate-limits)).
