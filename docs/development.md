# Development

```sh
make test        # go test -race ./...
make cover       # run tests and print a coverage summary
make vet         # go vet ./...
make fmt-check   # gofmt verification
make lint        # golangci-lint
make generate    # regenerate zz_generated_clients.go from services.json
make all         # fmt-check + vet + lint + test + build
```

`internal/awsx/registry` has a sanity test asserting known operations resolve as expected (`s3.ListBuckets`, `ec2.DescribeInstances`, ...); `internal/awsx/dispatch` and `internal/awsx/tools` test the generic invoke/read-only/error-mapping paths against a fake, reflectable client rather than real AWS calls, so the suite never touches a real account.

## Smoke-testing the protocol

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

## Adding a service

See [Services](services.md#adding-a-service).

## Releasing

Releases are tag-triggered (GoReleaser via CI, `.goreleaser.yaml`); `make next`/`make bump` compute and tag the next version from conventional commits.
