# CLI

`aws` is a small command tree (built on [`urfave/cli`](https://cli.urfave.org/)). A bare `aws` with no subcommand is equivalent to `aws mcp`.

## `aws mcp`

Run the MCP server over stdio. This is what MCP clients (Claude Desktop/Code, Cursor) invoke — see [Configuration](configuration.md) for client setup.

```sh
aws mcp
```

## `aws test`

Verify credentials against AWS: calls STS `GetCallerIdentity` for the active profile and prints the resolved principal. Useful for confirming credentials are correct before wiring up an MCP client.

```sh
$ aws test
OK  authenticated with AWS (region=us-west-2)
    account=123456789012 arn=arn:aws:iam::123456789012:user/ada
    read-only=false
```

## Next: browse the services

The server itself doesn't hard-code per-service commands — every AWS service is reachable generically through the [meta tools](index.md). See [Services](services.md) for what's currently registered, or [Architecture](architecture.md) for how discovery and dispatch work.
