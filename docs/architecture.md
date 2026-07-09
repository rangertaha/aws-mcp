# Architecture

aws-mcp does not hand-write a tool per AWS API call. It reflects over the AWS SDK v2 client types it's configured with, discovers every operation they expose, and dispatches calls to them generically. Adding a service is a one-line JSON entry, not a new package.

## Project layout

```
cmd/aws                entrypoint: a urfave/cli command tree (mcp, test)
internal/config         environment configuration (AWS_REGION, AWS_TOOLSETS, AWS_READONLY) + .env loading
internal/server         MCP server wrapper: typed tool registration, JSON Schema inference, read-only annotations, prompts
internal/awsx           AWS SDK v2 configuration and credentials
  manager.go              Manager: resolves aws.Config per profile, lazily builds/caches SDK clients
  profiles.go             ListProfiles: parses ~/.aws/config and ~/.aws/credentials
  check.go                Check: STS GetCallerIdentity connectivity check
internal/awsx/registry  the operation catalog, built by reflection
  reflect.go              discovers methods matching the SDK's operation shape
  classify.go             verb-prefix heuristic: mutating? destructive?
  unsupported.go          flags shapes generic JSON dispatch can't handle
  paginate.go             detects a continuation-token/marker output field
  zz_generated_clients.go generated: service name -> ClientFactory
internal/awsx/dispatch  generic invocation: decode -> reflect-call -> encode, plus AWS error mapping
internal/awsx/tools     the MCP tool surface built on registry + dispatch
internal/gen/services   code generator: services.json -> zz_generated_clients.go
internal/prompts        built-in MCP prompts (survey_bucket)
internal/app            wires config + awsx + tools + prompts into a *server.Server
```

## Building the catalog

At startup, `app.Assemble` filters `registry.Factories` (the generated service-name → `ClientFactory` map) down to whatever `AWS_TOOLSETS` allows, then calls `registry.Build` on that filtered set.

`registry.Build` constructs one throwaway instance of each configured client using a zero-value, unconfigured `aws.Config` — no credentials or network access needed, since this only inspects the client's *type*. For every exported method matching AWS SDK v2's standard operation shape —

```go
func(context.Context, *XInput, ...func(*Options)) (*XOutput, error)
```

— it records an `OperationSpec`: the operation's name, its `Input`/`Output` struct types (via `reflect.Type`, not generated code), whether it's *mutating* and *destructive* (a verb-prefix heuristic — `Get`/`List`/`Describe`/... are read-only, `Delete`/`Terminate`/... are destructive, and anything unrecognized defaults to mutating, since hiding a safe operation under read-only mode is an acceptable false positive but letting a mutating one through is not), whether it's *unsupported* by generic JSON dispatch (a streaming `io.Reader`/`io.Writer` field, an open-content "document" field, or a non-empty interface/union type encoding/json can't handle — see `s3.PutObject`'s body or `dynamodb`'s `AttributeValue`), and which output field (if any) carries a pagination token.

This is the whole discovery step: no per-service code, no codegen beyond the client constructor list, no maintenance as AWS ships new operations — the next SDK upgrade just adds them to the catalog.

## Calling an operation

An `aws_invoke` call with `{service, operation, input}` flows through `dispatch.Invoke`:

1. Look up the `OperationSpec` in the catalog. Unknown service/operation → error.
2. Refuse it if `Unsupported` (regardless of read-only mode) or if it's `Mutating` and the server is running read-only (`AWS_READONLY=true`).
3. Ask the `awsx.Manager` for a real SDK client for that service. The manager resolves `aws.Config` for the active profile via the standard credential chain (`awsconfig.LoadDefaultConfig`), calls the service's generated `ClientFactory`, and caches the result per `(profile, service)` so repeated calls don't re-resolve credentials or rebuild clients.
4. Reflectively allocate a new `*Input` struct (via `op.InputType`) and `json.Unmarshal` the caller's raw JSON into it.
5. Call the operation method by reflection: `reflect.ValueOf(client).MethodByName(operation).Call(...)`.
6. `json.Marshal` the `*Output` struct back out, or map a returned error through `smithy.APIError` into a structured `{code, message, fault}` — network/context errors pass through unchanged.

`aws_describe_operation` derives JSON Schema on demand for the same `Input`/`Output` types (via `jsonschema-go`), so a client can look up exactly what shape `aws_invoke` expects before calling it — no schema is precomputed or hand-written per operation.

## Profiles

`aws_list_profiles` reads section headers out of `~/.aws/config`/`~/.aws/credentials` (or the paths named by `AWS_CONFIG_FILE`/`AWS_SHARED_CREDENTIALS_FILE`). `aws_use_profile` switches the `Manager`'s active profile: it checks the name is one `aws_list_profiles` would discover, and eagerly resolves its static configuration (region, which credential source to use) so a typo'd or structurally invalid profile fails the call immediately rather than on the next `aws_invoke`. It does *not* verify the resulting credentials actually work — `LoadDefaultConfig` only wires up the credential provider chain without invoking it, so a profile with bogus static keys or an unauthenticated/expired SSO session still switches successfully; that failure only surfaces on the first real AWS call. Every subsequent `aws_invoke`/`aws_whoami` call uses whichever profile is currently active.

## Trade-off: binary size

Generic reflection-based dispatch has a real cost: the `aws` binary is around **670MB** uncompressed (~130MB in a released archive) — unusually large for a Go CLI. The cause is specific and measurable, not accidental bloat: `dispatch.Invoke` calls operations via `reflect.Value.Call` using a name resolved at runtime (`in.Operation`), so the Go linker cannot prove any given operation is unreachable and dead-code-eliminate its serializer/deserializer/endpoint-resolution code, the way it would for a hand-written client that only calls the specific operations it names directly. Measured in isolation: a trivial program that only imports `ec2.NewFromConfig` and never calls it links to ~5.6MB; the same program reflectively invoking every EC2 operation (mirroring `dispatch.Invoke`'s pattern) links to ~37MB — before counting the other 425 services. There is no code fix for this within the current design: retaining every operation generically dispatchable *is* the feature.

## Next: add a service, or browse what's covered

Adding a service is a one-line change, not a new package — see [Services](services.md#adding-a-service) for the exact steps and [Services](services.md) for the full, generated list of what's currently registered.
