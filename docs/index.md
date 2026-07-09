# aws-mcp

A [Model Context Protocol](https://modelcontextprotocol.io) (MCP) server, written
in Go, exposing AWS services as tools an LLM client can call. Built on
aws-sdk-go-v2.

Rather than hand-writing a tool per AWS API call, aws-mcp discovers every
operation on every configured AWS SDK v2 service client via reflection and
dispatches calls to them generically (see [Architecture](architecture.md)). A
handful of meta tools cover the entire AWS API surface — currently **426
services and 18,783 operations** (see [Services](services.md) for the full,
generated list):

| Tool                     | Purpose                                                              |
| ------------------------ | --------------------------------------------------------------------- |
| `aws_list_services`      | List every AWS service reachable through `aws_invoke`.                |
| `aws_list_operations`    | List a service's operations, flagging mutating/destructive/unsupported ones. |
| `aws_describe_operation` | Get an operation's JSON Schema (input and output) to build a call.    |
| `aws_invoke`             | Call any cataloged operation by service/operation name + JSON input.  |
| `aws_list_profiles`      | List AWS profiles from the shared config/credentials files.           |
| `aws_use_profile`        | Switch the active AWS profile.                                        |
| `aws_whoami`             | Verify credentials via STS GetCallerIdentity.                          |

## Next steps

- [Install](install.md) the server.
- Set up [Configuration](configuration.md), including your MCP client.
- Check the [CLI](cli.md) for `aws test`/`aws mcp`.
- Browse what's covered: [Services](services.md).
- Try a built-in [Prompt](prompts.md).
- Read how it works: [Architecture](architecture.md).
- Contributing? See [Development](development.md).
