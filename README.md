# aws-mcp

[![CI](https://github.com/rangertaha/aws-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/rangertaha/aws-mcp/actions/workflows/ci.yml)
[![Status: under construction](https://img.shields.io/badge/status-under%20construction-orange)](#-under-construction)
[![Go Reference](https://pkg.go.dev/badge/github.com/rangertaha/aws-mcp.svg)](https://pkg.go.dev/github.com/rangertaha/aws-mcp)
[![Go Version](https://img.shields.io/github/go-mod/go-version/rangertaha/aws-mcp)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

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
dispatches calls to them generically — **426 services, 18,783 operations**,
through a handful of meta tools:

| Tool                     | Purpose                                                              |
| ------------------------ | --------------------------------------------------------------------- |
| `aws_list_services`      | List every AWS service reachable through `aws_invoke`.                |
| `aws_list_operations`    | List a service's operations, flagging mutating/destructive/unsupported ones. |
| `aws_describe_operation` | Get an operation's JSON Schema (input and output) to build a call.    |
| `aws_invoke`             | Call any cataloged operation by service/operation name + JSON input.  |
| `aws_list_profiles`      | List AWS profiles from the shared config/credentials files.           |
| `aws_use_profile`        | Switch the active AWS profile.                                        |
| `aws_whoami`             | Verify credentials via STS GetCallerIdentity.                          |

**📖 Full documentation: [rangertaha.github.io/aws-mcp](https://rangertaha.github.io/aws-mcp/)** — install options, MCP client setup, the full 426-service list, architecture, and development guide all live there. This README only covers the quickstart.

## Quickstart

```sh
go install github.com/rangertaha/aws-mcp/cmd/aws@latest
aws test    # verify credentials (STS GetCallerIdentity)
aws mcp     # run the MCP server over stdio
```

See [Install](https://rangertaha.github.io/aws-mcp/install/) for prebuilt binaries and building from source. Note: the `aws` binary is unusually large (~670MB) — an inherent trade-off of generic reflection-based dispatch, explained there and in [Architecture](https://rangertaha.github.io/aws-mcp/architecture/).

Credentials come from the standard AWS chain (environment, `~/.aws`, SSO, or an attached IAM role) — nothing is stored by the server. Behavior is configured with:

| Variable       | Required | Description                                                  |
| -------------- | :------: | -------------------------------------------------------------- |
| `AWS_REGION`   |    no    | Region (standard AWS variable; also the override).             |
| `AWS_TOOLSETS` |    no    | Comma-separated AWS service names to enable, or `all`.         |
| `AWS_READONLY` |    no    | `true` to reject mutating operations (see `aws_list_operations`). |

See [Configuration](https://rangertaha.github.io/aws-mcp/configuration/) for MCP client setup (Claude Desktop/Code) and local development.

## Documentation

- [Install](https://rangertaha.github.io/aws-mcp/install/) — prebuilt binaries, `go install`, build from source.
- [Configuration](https://rangertaha.github.io/aws-mcp/configuration/) — environment variables, MCP client setup, local dev.
- [CLI](https://rangertaha.github.io/aws-mcp/cli/) — `aws mcp`, `aws test`.
- [Services](https://rangertaha.github.io/aws-mcp/services/) — the full, generated list of all 426 services and how to add one.
- [Prompts](https://rangertaha.github.io/aws-mcp/prompts/) — built-in guided workflows.
- [Architecture](https://rangertaha.github.io/aws-mcp/architecture/) — how discovery and dispatch work.
- [Development](https://rangertaha.github.io/aws-mcp/development/) — build, test, lint, smoke-test, release.

## Changelog

See [CHANGELOG.md](CHANGELOG.md).

## License

MIT — see [LICENSE](LICENSE).
