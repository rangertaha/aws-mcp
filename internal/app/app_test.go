// SPDX-License-Identifier: MIT

package app

import (
	"strings"
	"testing"

	"github.com/rangertaha/aws-mcp/internal/awsx/registry"
	"github.com/rangertaha/aws-mcp/internal/config"
)

func TestEnabledFactoriesFiltersKnownServices(t *testing.T) {
	got, err := enabledFactories(&config.Config{Toolsets: []string{"s3", "ec2"}})
	if err != nil {
		t.Fatalf("enabledFactories: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("enabledFactories() = %d services, want 2 (keys: %v)", len(got), keys(got))
	}
	if _, ok := got["s3"]; !ok {
		t.Errorf("expected s3 present, got %v", keys(got))
	}
	if _, ok := got["ec2"]; !ok {
		t.Errorf("expected ec2 present, got %v", keys(got))
	}
}

func TestEnabledFactoriesAll(t *testing.T) {
	got, err := enabledFactories(&config.Config{}) // empty Toolsets means "all"
	if err != nil {
		t.Fatalf("enabledFactories: %v", err)
	}
	if len(got) < 40 {
		t.Fatalf("enabledFactories with no filter = %d services, want the full catalog", len(got))
	}

	got2, err := enabledFactories(&config.Config{Toolsets: []string{"all"}})
	if err != nil {
		t.Fatalf("enabledFactories: %v", err)
	}
	if len(got2) != len(got) {
		t.Fatalf("enabledFactories(all) = %d, want %d (full catalog)", len(got2), len(got))
	}
}

// TestEnabledFactoriesRejectsUnknownEntries pins down the fail-fast behavior:
// a typo in AWS_TOOLSETS must produce a clear startup error naming the bad
// entry, not silently register fewer services than expected.
func TestEnabledFactoriesRejectsUnknownEntries(t *testing.T) {
	_, err := enabledFactories(&config.Config{Toolsets: []string{"s3", "not-a-real-service"}})
	if err == nil {
		t.Fatal("expected an error for an unrecognized AWS_TOOLSETS entry")
	}
	if !strings.Contains(err.Error(), "not-a-real-service") {
		t.Errorf("error = %q, want it to name the bad entry", err.Error())
	}
}

// TestEnabledFactoriesAllUnknownRejected guards specifically against the
// worst case: if every entry is a typo, the server must fail to start
// rather than silently coming up with zero services registered.
func TestEnabledFactoriesAllUnknownRejected(t *testing.T) {
	_, err := enabledFactories(&config.Config{Toolsets: []string{"nope", "also-nope"}})
	if err == nil {
		t.Fatal("expected an error when every AWS_TOOLSETS entry is unrecognized")
	}
}

// TestAssemble exercises Assemble itself, not just its enabledFactories
// helper: it builds no AWS clients and makes no network calls (registry.Build
// only reflects over an unconfigured aws.Config), so this runs hermetically
// with no credentials required.
func TestAssemble(t *testing.T) {
	srv, cleanup, err := Assemble(&config.Config{Toolsets: []string{"s3"}}, "test-version")
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if srv == nil {
		t.Fatal("Assemble returned a nil server")
	}
	if srv.ToolCount() == 0 {
		t.Error("ToolCount() = 0, want tools to be registered")
	}
	if srv.PromptCount() == 0 {
		t.Error("PromptCount() = 0, want prompts to be registered")
	}
	if got := srv.Toolsets(); len(got) != 1 || got[0] != "s3" {
		t.Errorf("Toolsets() = %v, want [s3]", got)
	}
	if srv.ReadOnly() {
		t.Error("ReadOnly() = true, want false for a zero-value Config")
	}

	if cleanup == nil {
		t.Fatal("Assemble returned a nil cleanup func")
	}
	cleanup() // must not panic
}

// TestAssembleReadOnly confirms the server's read-only policy actually
// reflects cfg.ReadOnly, not just its wiring being present.
func TestAssembleReadOnly(t *testing.T) {
	srv, _, err := Assemble(&config.Config{Toolsets: []string{"s3"}, ReadOnly: true}, "test-version")
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if !srv.ReadOnly() {
		t.Error("ReadOnly() = false, want true")
	}
}

// TestAssemblePropagatesEnabledFactoriesError confirms an unrecognized
// AWS_TOOLSETS entry fails Assemble itself (not just enabledFactories in
// isolation), and that the failure returns a nil server/cleanup rather than
// a half-built one.
func TestAssemblePropagatesEnabledFactoriesError(t *testing.T) {
	srv, cleanup, err := Assemble(&config.Config{Toolsets: []string{"not-a-real-service"}}, "test-version")
	if err == nil {
		t.Fatal("expected an error for an unrecognized AWS_TOOLSETS entry")
	}
	if srv != nil {
		t.Errorf("expected a nil server on error, got %v", srv)
	}
	if cleanup != nil {
		t.Error("expected a nil cleanup func on error")
	}
}

func keys(m map[string]registry.ClientFactory) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
