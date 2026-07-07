# aws-mcp

[![CI](https://github.com/rangertaha/aws-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/rangertaha/aws-mcp/actions/workflows/ci.yml)
[![Status: under construction](https://img.shields.io/badge/status-under%20construction-orange)](#-under-construction)

<div align="center">

## 🚧 &nbsp; UNDER CONSTRUCTION &nbsp; 🚧

**This server is a work in progress.**

APIs, configuration, and tool names may still change.

</div>

---

A [Model Context Protocol](https://modelcontextprotocol.io) (MCP) server, written
in Go, exposing **AWS** services as tools an LLM client (Claude Desktop/Code,
Cursor, and others) can call. Built on the official
[`aws-sdk-go-v2`](https://github.com/aws/aws-sdk-go-v2).

Rather than hand-writing a tool per AWS API call, aws-mcp discovers every
operation on every configured AWS SDK v2 service client via reflection and
dispatches calls to them generically. A handful of meta tools cover the
entire AWS API surface:

| Tool                     | Purpose                                                              |
| ------------------------ | --------------------------------------------------------------------- |
| `aws_list_services`      | List every AWS service reachable through `aws_invoke`.                |
| `aws_list_operations`    | List a service's operations, flagging mutating/destructive/unsupported ones. |
| `aws_describe_operation` | Get an operation's JSON Schema (input and output) to build a call.    |
| `aws_invoke`             | Call any cataloged operation by service/operation name + JSON input.  |
| `aws_list_profiles`      | List AWS profiles from the shared config/credentials files.           |
| `aws_use_profile`        | Switch the active AWS profile.                                        |
| `aws_whoami`             | Verify credentials via STS GetCallerIdentity.                          |

## Features

- **Generic dispatch**: every operation on every configured service is
  reachable without per-operation code — see `internal/awsx/registry` and
  `internal/awsx/dispatch`.
- **Schema discovery**: `aws_describe_operation` derives JSON Schema straight
  from the SDK's generated Input/Output structs.
- **Standard credential chain**: credentials come from the environment, shared
  config, SSO, or an attached IAM role — no secrets stored by the server.
- **Read-only switch**: `AWS_READONLY=true` rejects mutating operations at
  call time.
- **Service filtering**: enable only the services you need with `AWS_TOOLSETS`.

## Install

```sh
go install github.com/rangertaha/aws-mcp/cmd/aws@latest
```

Or build from source:

```sh
git clone https://github.com/rangertaha/aws-mcp
cd aws-mcp
make build        # produces ./bin/aws
```

## CLI

```sh
aws mcp      # run the MCP server over stdio (default when no subcommand)
aws test     # verify credentials (STS GetCallerIdentity)
```

## Configuration

Credentials follow the standard AWS chain. Server behavior is configured with:

| Variable       | Required | Description                                                  |
| -------------- | :------: | -------------------------------------------------------------- |
| `AWS_REGION`   |    no    | Region (standard AWS variable; also the override).             |
| `AWS_TOOLSETS` |    no    | Comma-separated AWS service names to enable, or `all`.         |
| `AWS_READONLY` |    no    | `true` to reject mutating operations (see `aws_list_operations`). |

## Adding a service

Every service in `internal/gen/services/services.json` is discovered and
exposed automatically — no per-service Go code is needed. To add one:

1. Add `"<name>": "github.com/aws/aws-sdk-go-v2/service/<name>"` to
   `internal/gen/services/services.json`.
2. Run `make generate` (regenerates `internal/awsx/registry/zz_generated_clients.go`).
3. `go mod tidy` to pick up the new SDK module.

## License

MIT — see [LICENSE](LICENSE).
