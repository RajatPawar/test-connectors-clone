# schemagen

One config-driven replacement for the per-provider scripts in `scripts/openapi/<provider>`. It uses
the same extraction (`api3`), and adds field writability from request bodies.

```sh
go run ./scripts/openapi/schemagen -config providers/<name>/metadata/schemagen.json
```

It writes `schemas.json` (V2 fields) into `output`. Then it prints a JSON report: the objects it
wrote, and a `problems` list with every endpoint it could not extract and why. It exits non-zero only
when the config or spec is unusable.

## Config

```jsonc
{
  "spec": "https://example.com/openapi.json",   // or a repo-relative path; Swagger 2.0 is converted
  "output": "providers/acme/metadata",
  "module": "root",                              // default root
  "hoistCommonPath": false,                      // true = common path prefix moves to the module path
  "mediaType": "application/json",               // e.g. application/vnd.api+json
  "require": ["contacts", "deals"],              // objects in scope — missing one fails the run
  "read": {
    "method": "GET",                             // or POST for search-style reads
    "allow": ["/contacts", "/deals*"],           // exact, prefix* or *suffix; allow = the whole list
    "deny": ["/ips"],
    "allowIdPaths": false,                       // keep paths with {params}
    "objects": {"/v2/people": "contacts"},       // path → object name (default: last segment)
    "displayNames": {"contacts": "Contacts"},
    "displayProcessors": ["camelCaseToSpaces", "capitalize"],  // + slashesToSpaces, pluralize
    "responseKey": {"default": "data", "objects": {"deals": "items"}},  // "@identical", "" = bare array
    "autoSelectArray": false,                    // take the only array-of-objects property
    "flatten": ["attributes"],                   // lift a nested object's fields (JSON:API)
    "onlyOptionalQueryParams": false,            // drop list endpoints needing a query param
    "stripPathPrefix": "/v1",                    // when the catalog BaseURL already has it
    "excludeFields": {"*": ["links"], "contacts": ["_embedded"]}
  },
  "write": {
    "contacts": {"create": "POST /contacts", "update": "PATCH /contacts/{id}",
                 "bodyPath": "data.attributes",  // fields inside a wrapped body (JSON:API)
                 "responsePath": "data"}         // same for the create response (write-only objects)
  },
  "fieldOverrides": {                            // corrections, applied last, visible in review
    "contacts": {"id": {"readOnly": true}, "score": {"valueType": "float"}}
  }
}
```

**Writability.** For each object under `write`, a field that the create or update request body
accepts is `readOnly: false`. A field that is only ever returned is `readOnly: true`. An object that
is written but never read gets its fields from the create's 2xx response. Without a `write` entry,
writability is unknown (`readOnly` unset). The one exception is a field the spec itself marks
`readOnly`.

**Types.** `number` becomes `float`, `date` and `date-time` become `date`/`datetime`, a string enum
becomes `singleSelect`, and an array of enum strings becomes `multiSelect`.

**Errors, not silent results.** The run fails, and changes nothing, in these cases:
- a `require`d object was not generated (the error names the reason api3 gave);
- a `bodyPath`/`responsePath` doesn't exist;
- an override or exclusion names a missing object or field;
- an unknown config key is used.

A config that no longer matches the spec fails loudly.

## What it does not do

These need a hand-written script, so they are out of scope. When a provider needs one, say so; don't
work around it:

- output in the V1 field format;
- custom per-object properties (e.g. zendesk pagination);
- path rewriting beyond stripping a prefix;
- combining GET and POST reads of the same object;
- hand-added fields, or specs that must be pre-processed before loading;
- specs api3 cannot parse.
