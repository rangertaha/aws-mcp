# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

Covers all **426 AWS services** shipped in aws-sdk-go-v2 (18,783 operations, 95.3% dispatchable).

### Added
- MCP server over stdio (aws-sdk-go-v2) that discovers every operation on every
  configured AWS service client via reflection and dispatches calls to them
  generically, instead of hand-writing a tool per API call: `aws_list_services`,
  `aws_list_operations`, `aws_describe_operation`, `aws_invoke`,
  `aws_list_profiles`, `aws_use_profile`, `aws_whoami`, a `survey_bucket`
  prompt, and an `aws test` credential check via STS GetCallerIdentity.
- `internal/gen/services/services.json` now registers all services present
  in aws-sdk-go-v2, not a curated subset — `make generate` regenerates the
  client factory registry from it. Kept current with upstream as new
  services ship (most recently `partnercentralrevenuemeasurement`).

### Fixed
- `json.RawMessage` fields (`aws_invoke`'s `input`/`output`, `aws_describe_operation`'s
  schema fields) were schema-inferred as a byte array (`{"type":"array","items":{"type":"integer"}}`)
  instead of an unconstrained "accept any JSON value" schema, which could mislead
  a strict client into sending the wrong shape entirely.
- Unsupported-operation detection now recurses into nested struct fields
  (with cycle protection) and checks unexported fields, instead of only the
  top-level fields of an Input/Output struct. This catches 325 more
  genuinely-broken operations (e.g. `dynamodb.TransactWriteItems`,
  `BatchWriteItem`, `TransactGetItems`) that were previously accepted by
  `aws_invoke` and could never actually succeed — any real payload failed
  with a raw `json: cannot unmarshal ...` error instead of a clean
  rejection (net +148 after also correctly un-flagging 177 operations; see
  below). Input and output are now checked asymmetrically: a
  union/interface field is only unsupported on the *input* side (JSON
  decoding can't populate one), since `json.Marshal` follows a populated
  interface's concrete value fine regardless of its static type — so 177
  operations whose only issue was an output-side union were correctly
  un-flagged as dispatchable.
- DynamoDB `Query`/`Scan` (and other List operations') pagination field
  went undetected: the pagination-field pattern listed `ExclusiveStartKey`
  (DynamoDB's *input* field name for the next page) but not
  `LastEvaluatedKey` (the actual *output* field), so `PaginationField` came
  back empty for some of the most commonly-paginated AWS operations.
- `aws_list_profiles` listed `[sso-session ...]` and `[services ...]`
  config-file sections as if they were selectable profiles; both are
  referenced *by* profiles, not profiles themselves, and switching to one
  via `aws_use_profile` would fail or behave unpredictably.
- An unrecognized `AWS_TOOLSETS` entry (e.g. a typo) now fails startup with
  a clear error naming it, instead of silently registering fewer services
  than expected — or, if every entry was unrecognized, zero services at
  all with no indication why.
- A race between a concurrent `aws_use_profile` call and an in-flight
  `aws_invoke`/client-build could cache an AWS SDK client under the
  *previous* profile's key while it was actually built with the *new*
  profile's credentials, so a later call under the old profile would
  silently reuse the wrong account's credentials.
- Doc-comments in `internal/server/server.go` and `internal/version.go`
  said "ado-mcp" (a copy-paste leftover from the sibling project used as a
  structural template) instead of "aws-mcp".
- `aws_describe_operation` errored outright for 55 real, commonly-used
  operations with a genuinely self-referential Go type (e.g. wafv2's
  `Statement`, nested via And/Or/Not sub-statements; organizations'
  `HandshakeResource`) — `jsonschema.ForType` has no way to express an
  unbounded type as a finite JSON Schema. These operations were never
  actually broken to call (`encoding/json` handles recursive struct types
  fine at the data level), only their schema was undiscoverable, making
  them effectively unusable without already knowing their shape from
  outside documentation. Every type that participates in a cycle is now
  found via reflection and overridden to a generic placeholder schema.
- `cloudwatchlogs.FilterLogEvents` — the primary CloudWatch Logs search API —
  was classified `Mutating: true` and so incorrectly rejected under
  `AWS_READONLY=true`, because `"Filter"` was missing from the read-verb
  prefix list (`"Search"`/`"Query"`/`"Scan"` were already there). It's the
  only `Filter*`-prefixed operation in the entire catalog and is purely
  read-only. Verified live against a real account.

### Documented
- The `aws` binary's size (~670MB uncompressed, ~130MB per released archive)
  was previously undocumented and would surprise anyone downloading a
  "prebuilt binary" expecting a normal-sized Go CLI. It's an inherent,
  measured consequence of generic reflection-based dispatch (retaining every
  operation's code since any could be invoked dynamically at runtime, not a
  fixable inefficiency) — now called out in the Install docs and explained
  in Architecture.
