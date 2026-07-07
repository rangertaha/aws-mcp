// SPDX-License-Identifier: MIT

package dispatch

import (
	"context"
	"encoding/json"
	"io"
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
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error = %T, want *APIError", err)
	}
	if apiErr.Code != "Boom" {
		t.Fatalf("Code = %q, want %q", apiErr.Code, "Boom")
	}
}

func TestInvokeDecodeErrorOnMalformedInput(t *testing.T) {
	cat, mgr := testCatalogAndManager(t)

	if _, err := Invoke(context.Background(), mgr, cat, "fake", "GetThing", json.RawMessage(`{not json`), false); err == nil {
		t.Fatal("expected a decode error for malformed input JSON")
	}
}
