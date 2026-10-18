# Orb

Read-only connector for Orb's billing API.

## Auth and base URL

API key from Orb > Developers > API keys, sent as `Authorization: Bearer <key>`.
The catalog base URL is `https://api.withorb.com`; the connector adds `/v1`.
A test-mode key reads the test-mode account.

## Objects

| Object          | Endpoint               | Notes                                  |
|-----------------|------------------------|----------------------------------------|
| `customers`     | `GET /v1/customers`    |                                        |
| `subscriptions` | `GET /v1/subscriptions`|                                        |
| `invoices`      | `GET /v1/invoices`     |                                        |
| `plans`         | `GET /v1/plans`        |                                        |
| `items`         | `GET /v1/items`        |                                        |
| `credit_notes`  | `GET /v1/credit_notes` |                                        |
| `coupons`       | `GET /v1/coupons`      | sent with `show_archived=true`         |

Records come back from the `data` array as Orb returns them. Nested objects (`customer`,
`billing_address`, `line_items` and so on) are kept as values of their top-level key.

## Pagination

Cursor-based. `limit` is 100 (Orb's maximum, the default is 20), and while
`pagination_metadata.has_more` is true, `pagination_metadata.next_cursor` is sent back as `cursor`.

## Incremental reads

None: every read is a full read and `Since`/`Until` are ignored. Orb records carry no
last-modified timestamp, and the list endpoints can only filter by creation (`created_at[...]`
on customers, plans and credit notes) or `invoice_date` (invoices). Items, coupons and
subscriptions take no time filter at all. A creation filter would miss records that change after
they are created (invoice status, voided credit notes, archived plans), so it is not used.

## Metadata

Customers, invoices, plans, items, credit notes and coupons come from a static catalog
(`internal/metadata/objectsMetadata.json`) transcribed from the response schemas on Orb's
fetch-object reference pages, including the documented enum values. Orb's published reference
has no schema for subscriptions, so their fields are sampled from the first record of
`GET /v1/subscriptions?limit=1`; an account with no subscriptions reports an error for that object.

## Rate limits

`GET` requests: 100 per second in live mode, 50 in test mode, 10 on trial accounts.
https://docs.withorb.com/api-reference/rate-limits
