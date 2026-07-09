// SPDX-License-Identifier: MIT

package tools

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"

	"github.com/rangertaha/aws-mcp/internal/awsx"
	"github.com/rangertaha/aws-mcp/internal/awsx/registry"
	"github.com/rangertaha/aws-mcp/internal/server"
)

// Options stands in for the SDK's per-operation functional-options type.
type Options struct{}

type GetThingInput struct {
	Name string `json:"name"`
}
type GetThingOutput struct {
	Value string `json:"value"`
}

type DeleteThingInput struct{}
type DeleteThingOutput struct{}

type fakeClient struct{}

func (c *fakeClient) GetThing(_ context.Context, in *GetThingInput, _ ...func(*Options)) (*GetThingOutput, error) {
	return &GetThingOutput{Value: "hello " + in.Name}, nil
}

func (c *fakeClient) DeleteThing(_ context.Context, _ *DeleteThingInput, _ ...func(*Options)) (*DeleteThingOutput, error) {
	return &DeleteThingOutput{}, nil
}

func testTools(readOnly bool) *tools {
	factories := map[string]registry.ClientFactory{
		"fake": func(awssdk.Config) any { return &fakeClient{} },
	}
	return &tools{
		mgr:      awsx.NewManager(factories, "", ""),
		cat:      registry.Build(factories),
		readOnly: readOnly,
	}
}

func TestListServices(t *testing.T) {
	svc := testTools(false)

	_, out, err := svc.listServices(context.Background(), nil, EmptyInput{})
	if err != nil {
		t.Fatalf("listServices: %v", err)
	}
	if out.Count != 1 || out.Items[0].Name != "fake" || out.Items[0].Operations != 2 {
		t.Fatalf("listServices() = %+v, want one service %q with 2 operations", out, "fake")
	}
}

func TestListOperations(t *testing.T) {
	svc := testTools(false)

	_, out, err := svc.listOperations(context.Background(), nil, ServiceInput{Service: "fake"})
	if err != nil {
		t.Fatalf("listOperations: %v", err)
	}
	if out.Count != 2 {
		t.Fatalf("listOperations() count = %d, want 2", out.Count)
	}
}

func TestListOperationsUnknownService(t *testing.T) {
	svc := testTools(false)

	if _, _, err := svc.listOperations(context.Background(), nil, ServiceInput{Service: "no-such-service"}); err == nil {
		t.Fatal("expected an error for an unknown service")
	}
}

func TestDescribeOperation(t *testing.T) {
	svc := testTools(false)

	_, desc, err := svc.describeOperation(context.Background(), nil, OperationInput{Service: "fake", Operation: "GetThing"})
	if err != nil {
		t.Fatalf("describeOperation: %v", err)
	}
	if len(desc.InputSchema) == 0 || len(desc.OutputSchema) == 0 {
		t.Fatalf("describeOperation() returned empty schema: %+v", desc)
	}
	var inSchema map[string]any
	if err := json.Unmarshal(desc.InputSchema, &inSchema); err != nil {
		t.Fatalf("input schema is not valid JSON: %v", err)
	}
}

func TestDescribeOperationUnknown(t *testing.T) {
	svc := testTools(false)

	if _, _, err := svc.describeOperation(context.Background(), nil, OperationInput{Service: "fake", Operation: "NoSuchOp"}); err == nil {
		t.Fatal("expected an error for an unknown operation")
	}
}

// TestDescribeOperationUnknownServiceIsDistinguishedFromUnknownOperation
// pins down that an unrecognized service name gets its own "unknown AWS
// service" error, not the generic "unknown AWS operation X.Y" message that
// would otherwise wrongly suggest the service exists.
func TestDescribeOperationUnknownServiceIsDistinguishedFromUnknownOperation(t *testing.T) {
	svc := testTools(false)

	_, _, err := svc.describeOperation(context.Background(), nil, OperationInput{Service: "no-such-service", Operation: "Whatever"})
	if err == nil {
		t.Fatal("expected an error for an unknown service")
	}
	if !strings.Contains(err.Error(), "unknown AWS service") {
		t.Errorf(`error = %q, want it to say "unknown AWS service", not "unknown AWS operation"`, err.Error())
	}
}

func TestInvoke(t *testing.T) {
	svc := testTools(false)

	_, out, err := svc.invoke(context.Background(), nil, InvokeInput{
		Service:   "fake",
		Operation: "GetThing",
		Input:     json.RawMessage(`{"name":"world"}`),
	})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	var got GetThingOutput
	if err := json.Unmarshal(out.Output, &got); err != nil {
		t.Fatalf("unmarshaling output: %v", err)
	}
	if got.Value != "hello world" {
		t.Fatalf("Value = %q, want %q", got.Value, "hello world")
	}
}

func TestInvokeHonorsReadOnly(t *testing.T) {
	svc := testTools(true)

	if _, _, err := svc.invoke(context.Background(), nil, InvokeInput{Service: "fake", Operation: "DeleteThing"}); err == nil {
		t.Fatal("expected a mutating operation to be rejected when readOnly=true")
	}
}

func TestListAndUseProfile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(dir, "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "credentials"))

	svc := testTools(false)

	_, profiles, err := svc.listProfiles(context.Background(), nil, EmptyInput{})
	if err != nil {
		t.Fatalf("listProfiles: %v", err)
	}
	if profiles.Count != 0 {
		t.Fatalf("listProfiles() = %+v, want none in an empty temp dir", profiles)
	}

	_, out, err := svc.useProfile(context.Background(), nil, ProfileInput{Profile: ""})
	if err != nil {
		t.Fatalf("useProfile(\"\"): %v", err)
	}
	if out.Profile != "" {
		t.Fatalf("useProfile(\"\").Profile = %q, want empty", out.Profile)
	}
}

func TestRegisterWiresUpTools(t *testing.T) {
	factories := map[string]registry.ClientFactory{
		"fake": func(awssdk.Config) any { return &fakeClient{} },
	}
	mgr := awsx.NewManager(factories, "", "")
	cat := registry.Build(factories)

	s := server.New("test", "0.0.0", false)
	Register(s, mgr, cat)

	const wantTools = 7 // list_services, list_operations, describe_operation, invoke, list_profiles, use_profile, whoami
	if s.ToolCount() != wantTools {
		t.Fatalf("ToolCount() = %d, want %d", s.ToolCount(), wantTools)
	}
	if toolsets := s.Toolsets(); len(toolsets) != 1 || toolsets[0] != "fake" {
		t.Fatalf("Toolsets() = %v, want [\"fake\"]", toolsets)
	}
}

func TestRegisterKeepsInvokeVisibleWhenReadOnly(t *testing.T) {
	factories := map[string]registry.ClientFactory{
		"fake": func(awssdk.Config) any { return &fakeClient{} },
	}
	mgr := awsx.NewManager(factories, "", "")
	cat := registry.Build(factories)

	s := server.New("test", "0.0.0", true)
	Register(s, mgr, cat)

	// aws_invoke enforces read-only per call (inside dispatch), not by being
	// hidden from a read-only server: mutating operations still need to be
	// describable/attempted so the caller gets a clear rejection.
	const wantTools = 7
	if s.ToolCount() != wantTools {
		t.Fatalf("ToolCount() = %d, want %d even when read-only", s.ToolCount(), wantTools)
	}
}
