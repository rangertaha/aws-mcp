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
[`aws-sdk-go-v2`](https://github.com/aws/aws-sdk-go-v2). See the
[**Documentation**](https://rangertaha.github.io/aws-mcp/) for more details.

**In this README**, roughly shallow to deep: [Install](#install) → [Features](#features) → [Configuration](#configuration) → [CLI](#cli) → [Services](#services) → [Prompts](#prompts-workflows) → [Architecture](#architecture) → [Development](#development) → [Changelog](#changelog). The [docs site](https://rangertaha.github.io/aws-mcp/) mirrors this same path as separate pages.

Rather than hand-writing a tool per AWS API call, aws-mcp discovers every
operation on every configured AWS SDK v2 service client via reflection and
dispatches calls to them generically. A handful of meta tools cover the
entire AWS API surface — **425 services and 18,765 operations** (see
[Services](docs/services.md) for the full, generated list):

| Tool                     | Purpose                                                              |
| ------------------------ | --------------------------------------------------------------------- |
| `aws_list_services`      | List every AWS service reachable through `aws_invoke`.                |
| `aws_list_operations`    | List a service's operations, flagging mutating/destructive/unsupported ones. |
| `aws_describe_operation` | Get an operation's JSON Schema (input and output) to build a call.    |
| `aws_invoke`             | Call any cataloged operation by service/operation name + JSON input.  |
| `aws_list_profiles`      | List AWS profiles from the shared config/credentials files.           |
| `aws_use_profile`        | Switch the active AWS profile.                                        |
| `aws_whoami`             | Verify credentials via STS GetCallerIdentity.                          |

## Install

Prebuilt binaries are published on the [latest release](https://github.com/rangertaha/aws-mcp/releases/latest). Download the archive for your platform, extract the `aws` binary, and put it on your `PATH`:

| Platform | Architecture          | Download (latest)                                                                                                            |
| -------- | --------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| macOS    | Apple Silicon (arm64) | [`aws-mcp_darwin_arm64.tar.gz`](https://github.com/rangertaha/aws-mcp/releases/latest/download/aws-mcp_darwin_arm64.tar.gz) |
| macOS    | Intel (amd64)         | [`aws-mcp_darwin_amd64.tar.gz`](https://github.com/rangertaha/aws-mcp/releases/latest/download/aws-mcp_darwin_amd64.tar.gz) |
| Linux    | amd64                 | [`aws-mcp_linux_amd64.tar.gz`](https://github.com/rangertaha/aws-mcp/releases/latest/download/aws-mcp_linux_amd64.tar.gz)   |
| Linux    | arm64                 | [`aws-mcp_linux_arm64.tar.gz`](https://github.com/rangertaha/aws-mcp/releases/latest/download/aws-mcp_linux_arm64.tar.gz)   |
| Windows  | amd64                 | [`aws-mcp_windows_amd64.zip`](https://github.com/rangertaha/aws-mcp/releases/latest/download/aws-mcp_windows_amd64.zip)     |
| Windows  | arm64                 | [`aws-mcp_windows_arm64.zip`](https://github.com/rangertaha/aws-mcp/releases/latest/download/aws-mcp_windows_arm64.zip)     |

Each link always resolves to the newest release. A [`checksums.txt`](https://github.com/rangertaha/aws-mcp/releases/latest/download/checksums.txt) is published alongside the archives.

<details>
<summary><strong>macOS / Linux</strong></summary>

Pick your `OS`/`ARCH`:

```sh
OS=darwin ARCH=arm64   # OS: darwin|linux   ARCH: amd64|arm64
curl -sSL "https://github.com/rangertaha/aws-mcp/releases/latest/download/aws-mcp_${OS}_${ARCH}.tar.gz" | tar -xz aws
sudo mv aws /usr/local/bin/
aws --version
```

</details>

<details>
<summary><strong>Windows (PowerShell)</strong></summary>

Pick your `$Arch`:

```powershell
$Arch = "amd64"   # ARCH: amd64|arm64
Invoke-WebRequest "https://github.com/rangertaha/aws-mcp/releases/latest/download/aws-mcp_windows_${Arch}.zip" -OutFile aws.zip
Expand-Archive aws.zip -DestinationPath .
.\aws.exe --version
```

</details>

<details>
<summary><strong>Install with Go</strong></summary>

```sh
go install github.com/rangertaha/aws-mcp/cmd/aws@latest
```

</details>

<details>
<summary><strong>Build from source</strong></summary>

```sh
git clone https://github.com/rangertaha/aws-mcp
cd aws-mcp
make build        # produces ./bin/aws
```

</details>

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

## Configuration

Credentials follow the standard AWS chain. Server behavior is configured with:

| Variable       | Required | Description                                                  |
| -------------- | :------: | -------------------------------------------------------------- |
| `AWS_REGION`   |    no    | Region (standard AWS variable; also the override).             |
| `AWS_TOOLSETS` |    no    | Comma-separated AWS service names to enable, or `all`. See [Services](#services). |
| `AWS_READONLY` |    no    | `true` to reject mutating operations (see `aws_list_operations`). |

### Use with Claude Desktop / Claude Code

Because credentials come from the standard chain, an MCP client config usually needs no secrets — just point it at the `aws` binary and, optionally, pick a profile/region:

```json
{
  "mcpServers": {
    "aws": {
      "command": "aws",
      "args": ["mcp"],
      "env": {
        "AWS_PROFILE": "your-profile",
        "AWS_REGION": "us-east-1"
      }
    }
  }
}
```

For Claude Code: `claude mcp add aws -- aws mcp` (add `--env AWS_PROFILE=...` for a non-default profile).

### Local development

The repo ships a committed [`.mcp.json`](.mcp.json) that runs the server straight from source (`go run ./cmd/aws mcp`), so changes take effect on the next session without a build step. Run `cp .env.example .env` and fill it in, or just rely on your existing `~/.aws` credentials.

## CLI

```sh
aws mcp      # run the MCP server over stdio (default when no subcommand)
aws test     # verify credentials (STS GetCallerIdentity)
```

## Services

<details>
<summary>All 425 services (18,765 operations)</summary>

Every service is registered in `internal/gen/services/services.json` and discovered automatically via reflection — nothing is hand-written per service. See [docs/services.md](docs/services.md) for the full table (service, operation count, unsupported count), or call `aws_list_services`/`aws_list_operations` at runtime for the live, authoritative answer.

### Adding a service

1. Add `"<name>": "github.com/aws/aws-sdk-go-v2/service/<name>"` to
   `internal/gen/services/services.json`.
2. Run `make generate` (regenerates `internal/awsx/registry/zz_generated_clients.go`).
3. `go mod tidy` to pick up the new SDK module.

No Go code to write — the new service's operations, schemas, and read-only classification are all derived automatically the next time the catalog is built.

</details>

## Prompts (workflows)

MCP clients surface **prompts** as slash commands automatically (e.g. in Claude Code and Claude Desktop). Built-in prompts:

| Prompt | Arguments | What it does |
| ------ | --------- | ------------ |
| `survey_bucket` | `bucket` | List an S3 bucket's objects via `aws_invoke` and summarize object count/size |

## Architecture

<details>
<summary>Project layout</summary>

```
cmd/aws                entrypoint: a urfave/cli command tree (mcp, test)
internal/config         environment configuration + .env loading
internal/server         MCP server wrapper: typed tool registration, JSON Schema inference, read-only policy, prompts
internal/awsx           AWS SDK v2 configuration/credentials: Manager (per-profile client cache), profile discovery, STS connectivity check
internal/awsx/registry  reflection-based operation catalog: discovery, mutating/destructive classification, unsupported-shape detection, pagination detection
internal/awsx/dispatch  generic invocation: JSON-decode input -> reflect-call the operation -> JSON-encode output, plus AWS error mapping
internal/awsx/tools     the MCP tool surface built on registry + dispatch
internal/gen/services   code generator: services.json -> zz_generated_clients.go (the only hand-maintained per-service artifact)
internal/prompts        built-in MCP prompts (survey_bucket)
internal/app            wires config + awsx + tools + prompts into a *server.Server
```

An `aws_invoke` call flows: look up the operation in the catalog → refuse it if unsupported or (mutating and read-only) → get a cached SDK client for the active profile from the `Manager` → reflectively decode the input, call the operation, encode the output (or map the error). See the [docs site's Architecture page](https://rangertaha.github.io/aws-mcp/architecture/) for the full request-flow walkthrough.

</details>

## Development

<details>
<summary>Build, test, and smoke-test</summary>

```sh
make test        # go test -race ./...
make cover       # run tests and print a coverage summary
make vet         # go vet ./...
make fmt-check   # gofmt verification
make lint        # golangci-lint
make generate    # regenerate zz_generated_clients.go from services.json
make all         # fmt-check + vet + lint + test + build
```

### Smoke-testing the protocol

List the tools over stdio without an MCP client:

```sh
printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"s","version":"0"}}}' \
  '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' \
| AWS_READONLY=true ./bin/aws mcp
```

Or browse interactively with the [MCP Inspector](https://github.com/modelcontextprotocol/inspector):

```sh
npx @modelcontextprotocol/inspector ./bin/aws mcp
```

Releases are tag-triggered (GoReleaser via CI); `make next`/`make bump` compute and tag the next version from conventional commits.

</details>

## Changelog

See [CHANGELOG.md](CHANGELOG.md).

## License

MIT — see [LICENSE](LICENSE).
