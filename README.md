# aws-mcp

[![CI](https://github.com/rangertaha/aws-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/rangertaha/aws-mcp/actions/workflows/ci.yml)
[![Status: under construction](https://img.shields.io/badge/status-under%20construction-orange)](#-under-construction)

<div align="center">

## 🚧 &nbsp; UNDER CONSTRUCTION &nbsp; 🚧

**This server is an early scaffold — a work in progress.**

It runs over stdio with **one read-only toolset** wired end-to-end.<br>
More toolsets are on the way (see the **TODO** list below).<br>
APIs, configuration, and tool names may still change.

</div>

---

A [Model Context Protocol](https://modelcontextprotocol.io) (MCP) server, written
in Go, exposing **AWS** services as tools an LLM client (Claude Desktop/Code,
Cursor, and others) can call. Built on the official
[`aws-sdk-go-v2`](https://github.com/aws/aws-sdk-go-v2).

## Features

- **Typed tools with schemas**: every tool has an auto-generated JSON Schema for
  its input and output, inferred from Go structs.
- **Standard credential chain**: credentials come from the environment, shared
  config, SSO, or an attached IAM role — no secrets stored by the server.
- **Read-only switch**: `AWS_READONLY=true` hides every mutating tool.
- **Toolset filtering**: enable only the services you need with `AWS_TOOLSETS`.

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

| Variable       | Required | Description                                            |
| -------------- | :------: | ------------------------------------------------------ |
| `AWS_REGION`   |    no    | Region (standard AWS variable; also the override).     |
| `AWS_TOOLSETS` |    no    | Comma-separated toolset names to enable, or `all`.     |
| `AWS_READONLY` |    no    | `true` to expose only read-only tools.                 |

## Toolsets

| Toolset | Covers                                                       |
| ------- | ------------------------------------------------------------ |
| `s3`    | list buckets (`s3_list_buckets`) and objects (`s3_list_objects`) |

### TODO toolsets

- `ec2` — describe instances, security groups, volumes.
- `iam` — list users, roles, and policies.
- `lambda` — list/get functions and configurations.
- `cloudwatch` — query metrics and logs.

> Each new service follows the same pattern: a client in `internal/aws/aws.go`,
> a package under `internal/aws/<service>/`, and an entry in
> `internal/app/app.go`.

## License

MIT — see [LICENSE](LICENSE).
