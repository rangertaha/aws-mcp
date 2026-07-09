// SPDX-License-Identifier: MIT

package dispatch

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/smithy-go"

	"github.com/rangertaha/aws-mcp/internal/awsx"
	"github.com/rangertaha/aws-mcp/internal/awsx/registry"
)

// Options stands in for the SDK's per-operation functional-options type; its
// contents don't matter to reflection-based discovery, only its shape
// (variadic ...func(*Options)) does.
type Options struct{}

type GetThingInput struct {
	Name string `json:"name"`
}
type GetThingOutput struct {
	Value string `json:"value"`
}

type DeleteThingInput struct {
	Name string `json:"name"`
}
type DeleteThingOutput struct {
	Deleted bool `json:"deleted"`
}

type FailOpInput struct{}
type FailOpOutput struct{}

type PanicOpInput struct{}
type PanicOpOutput struct{}

type PanicNilOpInput struct{}
type PanicNilOpOutput struct{}

type PanicErrorOpInput struct{}
type PanicErrorOpOutput struct{}

// BadOpInput carries a streaming field, which registry.unsupported flags as
// Unsupported: Invoke must refuse to dispatch it regardless of readOnly.
type BadOpInput struct {
	Body io.Reader
}
type BadOpOutput struct{}

type fakeClient struct{}

func (c *fakeClient) GetThing(_ context.Context, in *GetThingInput, _ ...func(*Options)) (*GetThingOutput, error) {
	return &GetThingOutput{Value: "hello " + in.Name}, nil
}

func (c *fakeClient) DeleteThing(_ context.Context, _ *DeleteThingInput, _ ...func(*Options)) (*DeleteThingOutput, error) {
	return &DeleteThingOutput{Deleted: true}, nil
}

func (c *fakeClient) FailOp(_ context.Context, _ *FailOpInput, _ ...func(*Options)) (*FailOpOutput, error) {
	return nil, &smithy.GenericAPIError{Code: "Boom", Message: "it broke", Fault: smithy.FaultServer}
}

func (c *fakeClient) BadOp(_ context.Context, _ *BadOpInput, _ ...func(*Options)) (*BadOpOutput, error) {
	return &BadOpOutput{}, nil
}

// PanicOp stands in for a hypothetical AWS SDK v2 bug that panics on some
// edge-case input, deep inside a real operation's serialization/middleware
// stack — something dispatch has no way to prevent, only contain.
func (c *fakeClient) PanicOp(_ context.Context, _ *PanicOpInput, _ ...func(*Options)) (*PanicOpOutput, error) {
	panic("simulated SDK-internal panic")
}

// PanicNilOp and PanicErrorOp pin down that safeCall's "r != nil" recover
// check isn't fooled by the two edge cases a plain string panic doesn't
// exercise: panic(nil) (which, since Go 1.21, recover()s as a non-nil
// *runtime.PanicNilError rather than being silently swallowed) and a
// panic with a non-string value (a common real pattern, e.g.
// panic(fmt.Errorf(...))), which %v must still format sensibly.
func (c *fakeClient) PanicNilOp(_ context.Context, _ *PanicNilOpInput, _ ...func(*Options)) (*PanicNilOpOutput, error) {
	panic(nil)
}

func (c *fakeClient) PanicErrorOp(_ context.Context, _ *PanicErrorOpInput, _ ...func(*Options)) (*PanicErrorOpOutput, error) {
	panic(errors.New("simulated error-typed panic"))
}

// testCatalogAndManager builds a registry.Catalog and awsx.Manager backed by
// a single fake service, so Invoke can be exercised without any real AWS
// client or network access.
func testCatalogAndManager(t *testing.T) (*registry.Catalog, *awsx.Manager) {
	t.Helper()
	factories := map[string]registry.ClientFactory{
		"fake": func(awssdk.Config) any { return &fakeClient{} },
	}
	return registry.Build(factories), awsx.NewManager(factories, "", "")
}

func TestInvokeSuccess(t *testing.T) {
	cat, mgr := testCatalogAndManager(t)

	out, err := Invoke(context.Background(), mgr, cat, "fake", "GetThing", json.RawMessage(`{"name":"world"}`), false)
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}

	var got GetThingOutput
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshaling output: %v", err)
	}
	if got.Value != "hello world" {
		t.Fatalf("Value = %q, want %q", got.Value, "hello world")
	}
}

func TestInvokeEmptyInput(t *testing.T) {
	cat, mgr := testCatalogAndManager(t)

	out, err := Invoke(context.Background(), mgr, cat, "fake", "GetThing", nil, false)
	if err != nil {
		t.Fatalf("Invoke with nil input: %v", err)
	}
	var got GetThingOutput
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshaling output: %v", err)
	}
	if got.Value != "hello " {
		t.Fatalf("Value = %q, want %q", got.Value, "hello ")
	}
}

func TestInvokeUnknownOperation(t *testing.T) {
	cat, mgr := testCatalogAndManager(t)

	if _, err := Invoke(context.Background(), mgr, cat, "fake", "NoSuchOp", nil, false); err == nil {
		t.Fatal("expected an error for an unknown operation")
	}
}

// TestInvokeUnknownServiceIsDistinguishedFromUnknownOperation pins down that
// an unrecognized service name gets its own error naming the service, rather
// than the generic "unknown operation X.Y" message that would otherwise
// wrongly suggest the service exists but the operation name is the problem.
func TestInvokeUnknownServiceIsDistinguishedFromUnknownOperation(t *testing.T) {
	cat, mgr := testCatalogAndManager(t)

	_, err := Invoke(context.Background(), mgr, cat, "no-such-service", "Whatever", nil, false)
	if err == nil {
		t.Fatal("expected an error for an unknown service")
	}
	if !strings.Contains(err.Error(), "unknown AWS service") {
		t.Errorf(`error = %q, want it to say "unknown AWS service", not "unknown operation"`, err.Error())
	}
}

func TestInvokeUnsupportedOperation(t *testing.T) {
	cat, mgr := testCatalogAndManager(t)

	if _, err := Invoke(context.Background(), mgr, cat, "fake", "BadOp", nil, false); err == nil {
		t.Fatal("expected an error for an unsupported operation")
	}
}

func TestInvokeRejectsMutatingWhenReadOnly(t *testing.T) {
	cat, mgr := testCatalogAndManager(t)

	if _, err := Invoke(context.Background(), mgr, cat, "fake", "DeleteThing", json.RawMessage(`{"name":"x"}`), true); err == nil {
		t.Fatal("expected DeleteThing to be rejected under readOnly=true")
	}
}

func TestInvokeAllowsMutatingWhenNotReadOnly(t *testing.T) {
	cat, mgr := testCatalogAndManager(t)

	out, err := Invoke(context.Background(), mgr, cat, "fake", "DeleteThing", json.RawMessage(`{"name":"x"}`), false)
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	var got DeleteThingOutput
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshaling output: %v", err)
	}
	if !got.Deleted {
		t.Fatalf("Deleted = false, want true")
	}
}

func TestInvokeAllowsReadOnlyOperationWhenReadOnly(t *testing.T) {
	cat, mgr := testCatalogAndManager(t)

	if _, err := Invoke(context.Background(), mgr, cat, "fake", "GetThing", json.RawMessage(`{"name":"x"}`), true); err != nil {
		t.Fatalf("expected GetThing to be allowed under readOnly=true, got: %v", err)
	}
}

func TestInvokeMapsAPIError(t *testing.T) {
	cat, mgr := testCatalogAndManager(t)

	_, err := Invoke(context.Background(), mgr, cat, "fake", "FailOp", nil, false)
	if err == nil {
		t.Fatal("expected an error from FailOp")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %T, want *APIError", err)
	}
	if apiErr.Code != "Boom" {
		t.Fatalf("Code = %q, want %q", apiErr.Code, "Boom")
	}
}

// TestInvokeRecoversPanicFromDispatchedOperation pins down that a panic
// occurring inside the reflectively-invoked operation itself (simulating an
// AWS SDK v2 internal bug on some edge-case input) is contained as a clean
// error return, not left to unwind the goroutine — which, uncaught, would
// crash the entire MCP server process and every other in-flight or future
// request along with it. Neither aws-mcp nor the vendored MCP SDK recovers
// panics anywhere else in the call path, so this is the only backstop.
func TestInvokeRecoversPanicFromDispatchedOperation(t *testing.T) {
	cat, mgr := testCatalogAndManager(t)

	_, err := Invoke(context.Background(), mgr, cat, "fake", "PanicOp", nil, false)
	if err == nil {
		t.Fatal("expected an error, not a successful return, when the dispatched operation panics")
	}
	if !strings.Contains(err.Error(), "simulated SDK-internal panic") {
		t.Errorf("error = %q, want it to mention the recovered panic value", err.Error())
	}
}

func TestInvokeRecoversPanicWithNonStringValue(t *testing.T) {
	cat, mgr := testCatalogAndManager(t)

	_, err := Invoke(context.Background(), mgr, cat, "fake", "PanicErrorOp", nil, false)
	if err == nil {
		t.Fatal("expected an error when the dispatched operation panics with a non-string (error) value")
	}
	if !strings.Contains(err.Error(), "simulated error-typed panic") {
		t.Errorf("error = %q, want it to mention the recovered panic value", err.Error())
	}
}

func TestInvokeRecoversPanicNil(t *testing.T) {
	cat, mgr := testCatalogAndManager(t)

	_, err := Invoke(context.Background(), mgr, cat, "fake", "PanicNilOp", nil, false)
	if err == nil {
		t.Fatal("expected an error when the dispatched operation calls panic(nil), not a successful return")
	}
}

func TestInvokeDecodeErrorOnMalformedInput(t *testing.T) {
	cat, mgr := testCatalogAndManager(t)

	if _, err := Invoke(context.Background(), mgr, cat, "fake", "GetThing", json.RawMessage(`{not json`), false); err == nil {
		t.Fatal("expected a decode error for malformed input JSON")
	}
}
