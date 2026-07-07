# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

Covers all **425 AWS services** shipped in aws-sdk-go-v2 (18,765 operations, 96.1% dispatchable).

### Added
- MCP server over stdio (aws-sdk-go-v2) that discovers every operation on every
  configured AWS service client via reflection and dispatches calls to them
  generically, instead of hand-writing a tool per API call: `aws_list_services`,
  `aws_list_operations`, `aws_describe_operation`, `aws_invoke`,
  `aws_list_profiles`, `aws_use_profile`, `aws_whoami`, a `survey_bucket`
  prompt, and an `aws test` credential check via STS GetCallerIdentity.
- `internal/gen/services/services.json` now registers all 425 services present
  in aws-sdk-go-v2, not a curated subset — `make generate` regenerates the
  client factory registry from it.

### Fixed
- `json.RawMessage` fields (`aws_invoke`'s `input`/`output`, `aws_describe_operation`'s
  schema fields) were schema-inferred as a byte array (`{"type":"array","items":{"type":"integer"}}`)
  instead of an unconstrained "accept any JSON value" schema, which could mislead
  a strict client into sending the wrong shape entirely.
