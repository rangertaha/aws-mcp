# Configuration

Credentials come from the standard AWS credential chain (environment variables,
`~/.aws/config` & `~/.aws/credentials`, SSO, or an attached IAM role). aws-mcp
does not store credentials. Server behavior is configured with:

| Variable       | Required | Description                                          |
| -------------- | :------: | ---------------------------------------------------- |
| `AWS_REGION`   |    no    | Region (standard AWS variable; also the override).   |
| `AWS_TOOLSETS` |    no    | Comma-separated toolset names to enable, or `all`.   |
| `AWS_READONLY` |    no    | `true` to expose only read-only tools.               |
