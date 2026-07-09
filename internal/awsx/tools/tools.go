// SPDX-License-Identifier: MIT

// Package tools registers the generic, model-facing MCP tools that expose
// every AWS SDK v2 operation discovered by package registry, dispatched
// generically by package dispatch. Unlike a hand-written per-service
// toolset, this is the only toolset aws-mcp registers: adding a new AWS
// service means adding an entry to internal/gen/services/services.json, not
// writing new tools.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rangertaha/aws-mcp/internal/awsx"
	"github.com/rangertaha/aws-mcp/internal/awsx/dispatch"
	"github.com/rangertaha/aws-mcp/internal/awsx/registry"
	"github.com/rangertaha/aws-mcp/internal/server"
)

// Register adds the generic AWS discovery/invocation tools to s, backed by
// mgr (client configuration/caching) and cat (the discovered operation
// catalog). It also notes one toolset per cataloged service so
// server.Toolsets reports what's actually reachable.
func Register(s *server.Server, mgr *awsx.Manager, cat *registry.Catalog) {
	for _, name := range cat.ServiceNames() {
		s.NoteToolset(name)
	}

	svc := &tools{mgr: mgr, cat: cat, readOnly: s.ReadOnly()}

	server.Register(s, server.ToolDef{
		Name:        "aws_list_services",
		Title:       "List AWS services",
		Description: "List every AWS service reachable through aws_invoke, with its operation count.",
	}, svc.listServices)

	server.Register(s, server.ToolDef{
		Name:        "aws_list_operations",
		Title:       "List AWS operations",
		Description: "List the operations available on one AWS service, noting whether each mutates AWS state and whether generic dispatch supports it.",
	}, svc.listOperations)

	server.Register(s, server.ToolDef{
		Name:        "aws_describe_operation",
		Title:       "Describe an AWS operation",
		Description: "Describe one AWS operation's input/output JSON schema, to construct a valid aws_invoke call.",
	}, svc.describeOperation)

	server.Register(s, server.ToolDef{
		Name:  "aws_invoke",
		Title: "Invoke an AWS operation",
		Description: "Call any cataloged AWS operation by service and operation name, with a JSON input matching its schema " +
			"(see aws_describe_operation). Mutating operations are rejected when the server is running read-only.",
	}, svc.invoke)

	server.Register(s, server.ToolDef{
		Name:        "aws_list_profiles",
		Title:       "List AWS profiles",
		Description: "List the AWS profiles discovered in the shared config and credentials files.",
	}, svc.listProfiles)

	server.Register(s, server.ToolDef{
		Name:        "aws_use_profile",
		Title:       "Switch AWS profile",
		Description: "Switch the AWS profile used by subsequent aws_invoke calls, verifying it resolves before switching.",
	}, svc.useProfile)

	server.Register(s, server.ToolDef{
		Name:        "aws_whoami",
		Title:       "Check AWS identity",
		Description: "Verify credentials for the active profile via STS GetCallerIdentity and return the resolved principal.",
	}, svc.whoami)
}

type tools struct {
	mgr      *awsx.Manager
	cat      *registry.Catalog
	readOnly bool
}

// EmptyInput is used by tools that take no arguments.
type EmptyInput struct{}

// ServiceSummary describes one cataloged AWS service.
type ServiceSummary struct {
	Name       string `json:"name" jsonschema:"AWS service name, e.g. s3"`
	Operations int    `json:"operations" jsonschema:"number of operations discovered for this service"`
}

func (t *tools) listServices(_ context.Context, _ *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, server.ListResult[ServiceSummary], error) {
	names := t.cat.ServiceNames()
	out := make([]ServiceSummary, 0, len(names))
	for _, name := range names {
		svc, _ := t.cat.Service(name)
		out = append(out, ServiceSummary{Name: name, Operations: len(svc.Operations)})
	}
	return nil, server.List(out), nil
}

// ServiceInput identifies a single cataloged AWS service.
type ServiceInput struct {
	Service string `json:"service" jsonschema:"AWS service name, e.g. s3 (see aws_list_services)"`
}

// OperationSummary describes one cataloged AWS operation.
type OperationSummary struct {
	Name              string `json:"name" jsonschema:"operation name, e.g. ListBuckets"`
	Mutating          bool   `json:"mutating" jsonschema:"whether the operation is believed to change AWS state"`
	Destructive       bool   `json:"destructive" jsonschema:"whether the operation deletes, replaces, or disables resources"`
	Unsupported       bool   `json:"unsupported" jsonschema:"whether generic dispatch can safely call this operation"`
	UnsupportedReason string `json:"unsupportedReason,omitempty" jsonschema:"why dispatch can't call this operation, if unsupported"`
	PaginationField   string `json:"paginationField,omitempty" jsonschema:"output field carrying a continuation token/marker, if any"`
}

func (t *tools) listOperations(_ context.Context, _ *mcp.CallToolRequest, in ServiceInput) (*mcp.CallToolResult, server.ListResult[OperationSummary], error) {
	svc, ok := t.cat.Service(in.Service)
	if !ok {
		return nil, server.ListResult[OperationSummary]{}, fmt.Errorf("unknown AWS service %q", in.Service)
	}

	names := svc.OperationNames()
	out := make([]OperationSummary, 0, len(names))
	for _, name := range names {
		op := svc.Operations[name]
		out = append(out, OperationSummary{
			Name:              op.Name,
			Mutating:          op.Mutating,
			Destructive:       op.Destructive,
			Unsupported:       op.Unsupported,
			UnsupportedReason: op.UnsupportedReason,
			PaginationField:   op.PaginationField,
		})
	}
	return nil, server.List(out), nil
}

// OperationInput identifies a single operation on a cataloged AWS service.
type OperationInput struct {
	Service   string `json:"service" jsonschema:"AWS service name, e.g. s3"`
	Operation string `json:"operation" jsonschema:"operation name, e.g. ListBuckets (see aws_list_operations)"`
}

// OperationDetail describes one AWS operation's schema and dispatch metadata.
type OperationDetail struct {
	OperationSummary
	InputSchema  json.RawMessage `json:"inputSchema" jsonschema:"JSON Schema for the aws_invoke input argument"`
	OutputSchema json.RawMessage `json:"outputSchema" jsonschema:"JSON Schema for the aws_invoke output"`
}

func (t *tools) describeOperation(_ context.Context, _ *mcp.CallToolRequest, in OperationInput) (*mcp.CallToolResult, OperationDetail, error) {
	if _, ok := t.cat.Service(in.Service); !ok {
		return nil, OperationDetail{}, fmt.Errorf("unknown AWS service %q", in.Service)
	}
	op, ok := t.cat.Operation(in.Service, in.Operation)
	if !ok {
		return nil, OperationDetail{}, fmt.Errorf("unknown AWS operation %s.%s", in.Service, in.Operation)
	}

	inSchema, err := operationSchema(op.InputType)
	if err != nil {
		return nil, OperationDetail{}, fmt.Errorf("building input schema for %s.%s: %w", in.Service, in.Operation, err)
	}
	outSchema, err := operationSchema(op.OutputType)
	if err != nil {
		return nil, OperationDetail{}, fmt.Errorf("building output schema for %s.%s: %w", in.Service, in.Operation, err)
	}

	return nil, OperationDetail{
		OperationSummary: OperationSummary{
			Name:              op.Name,
			Mutating:          op.Mutating,
			Destructive:       op.Destructive,
			Unsupported:       op.Unsupported,
			UnsupportedReason: op.UnsupportedReason,
			PaginationField:   op.PaginationField,
		},
		InputSchema:  inSchema,
		OutputSchema: outSchema,
	}, nil
}

// operationSchema builds the JSON schema for an AWS SDK Input/Output struct
// type. IgnoreInvalidTypes covers Unsupported operations too (e.g. streaming
// bodies), so aws_describe_operation can still document why a field is
// missing rather than erroring outright.
//
// Some AWS types are genuinely self-referential (e.g. wafv2's Statement,
// which nests itself via And/Or/Not sub-statements; or organizations'
// HandshakeResource) — jsonschema.ForType has no way to express that as a
// finite JSON Schema and errors out entirely. encoding/json handles these
// fine at the data level (real values have finite depth even though the Go
// type doesn't); only schema *generation* needs an escape hatch, so every
// type that actually participates in a cycle is pre-identified via
// reflection and overridden to a generic placeholder schema, breaking the
// cycle before jsonschema-go's own cycle detection would otherwise error.
func operationSchema(t reflect.Type) (json.RawMessage, error) {
	opts := &jsonschema.ForOptions{
		IgnoreInvalidTypes: true,
		TypeSchemas:        cyclicTypeOverrides(t),
	}
	s, err := jsonschema.ForType(t, opts)
	if err != nil {
		return nil, err
	}
	return json.Marshal(s)
}

// cyclicTypeOverrides finds every named struct type reachable from t
// (through pointers, slices, arrays, and maps) that participates in a
// self-reference — directly or through other types — and returns a schema
// override for each so jsonschema.ForType treats it as an opaque object
// instead of trying to expand it indefinitely. Returns nil if t has no
// cyclic types, so ForOptions.TypeSchemas stays nil (its natural zero value).
func cyclicTypeOverrides(t reflect.Type) map[reflect.Type]*jsonschema.Schema {
	cyclic := map[reflect.Type]bool{}
	findCyclicTypes(t, map[reflect.Type]bool{}, cyclic)
	if len(cyclic) == 0 {
		return nil
	}
	overrides := make(map[reflect.Type]*jsonschema.Schema, len(cyclic))
	for ct := range cyclic {
		overrides[ct] = &jsonschema.Schema{Description: "recursive type (self-referential); see the AWS API reference for its structure"}
	}
	return overrides
}

// findCyclicTypes walks t's type graph (not its values — this is purely
// structural, so it terminates even though the type itself may recurse
// indefinitely) via path, the set of named struct types on the current
// recursion stack. A type reached while already on that stack is added to
// cyclic.
func findCyclicTypes(t reflect.Type, path map[reflect.Type]bool, cyclic map[reflect.Type]bool) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		findCyclicTypes(t.Elem(), path, cyclic)
	case reflect.Struct:
		if path[t] {
			cyclic[t] = true
			return
		}
		path[t] = true
		defer delete(path, t)
		for i := 0; i < t.NumField(); i++ {
			findCyclicTypes(t.Field(i).Type, path, cyclic)
		}
	}
}

// InvokeInput names an operation to call and its JSON input.
type InvokeInput struct {
	Service   string          `json:"service" jsonschema:"AWS service name, e.g. s3"`
	Operation string          `json:"operation" jsonschema:"operation name, e.g. ListBuckets"`
	Input     json.RawMessage `json:"input,omitempty" jsonschema:"JSON input matching the operation's input schema (see aws_describe_operation); omit for operations that take no fields"`
}

// InvokeOutput carries the raw JSON result of an aws_invoke call.
type InvokeOutput struct {
	Output json.RawMessage `json:"output" jsonschema:"JSON output matching the operation's output schema"`
}

func (t *tools) invoke(ctx context.Context, _ *mcp.CallToolRequest, in InvokeInput) (*mcp.CallToolResult, InvokeOutput, error) {
	out, err := dispatch.Invoke(ctx, t.mgr, t.cat, in.Service, in.Operation, in.Input, t.readOnly)
	if err != nil {
		return nil, InvokeOutput{}, err
	}
	return nil, InvokeOutput{Output: out}, nil
}

func (t *tools) listProfiles(_ context.Context, _ *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, server.ListResult[string], error) {
	names, err := awsx.ListProfiles()
	if err != nil {
		return nil, server.ListResult[string]{}, err
	}
	return nil, server.List(names), nil
}

// ProfileInput names an AWS profile.
type ProfileInput struct {
	Profile string `json:"profile" jsonschema:"AWS profile name to switch to (see aws_list_profiles); empty reverts to the default credential chain"`
}

// ProfileOutput reports the now-active AWS profile.
type ProfileOutput struct {
	Profile string `json:"profile" jsonschema:"the now-active AWS profile"`
}

func (t *tools) useProfile(ctx context.Context, _ *mcp.CallToolRequest, in ProfileInput) (*mcp.CallToolResult, ProfileOutput, error) {
	if err := t.mgr.UseProfile(ctx, in.Profile); err != nil {
		return nil, ProfileOutput{}, err
	}
	return nil, ProfileOutput{Profile: t.mgr.Profile()}, nil
}

func (t *tools) whoami(ctx context.Context, _ *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, awsx.Identity, error) {
	id, err := awsx.Check(ctx, t.mgr)
	if err != nil {
		return nil, awsx.Identity{}, err
	}
	return nil, *id, nil
}
