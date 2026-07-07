# Configuration

Credentials come from the standard AWS credential chain (environment variables,
`~/.aws/config` & `~/.aws/credentials`, SSO, or an attached IAM role). aws-mcp
does not store credentials. Server behavior is configured with:

| Variable       | Required | Description                                                |
| -------------- | :------: | -------------------------------------------------------------- |
| `AWS_REGION`   |    no    | Region (standard AWS variable; also the override).          |
| `AWS_TOOLSETS` |    no    | Comma-separated AWS service names to enable, or `all`. See [Services](services.md) for valid names. |
| `AWS_READONLY` |    no    | `true` to reject mutating operations at call time.          |

## Use with Claude Desktop / Claude Code

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

For Claude Code: `claude mcp add aws -- aws mcp` (add `--env AWS_PROFILE=...` if you need a non-default profile).

## Local development

The repo ships a committed [`.mcp.json`](.mcp.json) that runs the server straight from source (`go run ./cmd/aws mcp`), so changes take effect on the next session without a build step. Run `cp .env.example .env` and fill it in (or just rely on your existing `~/.aws` credentials) before launching Claude Code in this directory.
