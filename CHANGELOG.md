# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial scaffold: MCP server over stdio (aws-sdk-go-v2) with the `s3` toolset
  (`s3_list_buckets`, `s3_list_objects`), a `survey_bucket` prompt, and an
  `aws test` credential check via STS GetCallerIdentity.
