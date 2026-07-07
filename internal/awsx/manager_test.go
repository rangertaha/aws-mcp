// SPDX-License-Identifier: MIT

package awsx

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"

	"github.com/rangertaha/aws-mcp/internal/awsx/registry"
)

type fakeClient struct{ id int }

func countingFactory(calls *int) registry.ClientFactory {
	return func(awssdk.Config) any {
		*calls++
		return &fakeClient{id: *calls}
	}
}

func TestManagerClientCachesPerProfile(t *testing.T) {
	var calls int
	mgr := NewManager(map[string]registry.ClientFactory{"fake": countingFactory(&calls)}, "", "")

	first, err := mgr.Client(context.Background(), "fake")
	if err != nil {
		t.Fatalf("Client: %v", err)
	}
	second, err := mgr.Client(context.Background(), "fake")
	if err != nil {
		t.Fatalf("Client: %v", err)
	}

	if calls != 1 {
		t.Fatalf("factory called %d times, want 1", calls)
	}
	if first.(*fakeClient) != second.(*fakeClient) {
		t.Fatalf("Client returned different instances across calls")
	}
}

func TestManagerClientUnknownService(t *testing.T) {
	mgr := NewManager(map[string]registry.ClientFactory{}, "", "")

	if _, err := mgr.Client(context.Background(), "no-such-service"); err == nil {
		t.Fatal("expected an error for an unregistered service")
	}
}

func TestManagerProfileDefaultsToConstructorArg(t *testing.T) {
	mgr := NewManager(nil, "", "staging")
	if got := mgr.Profile(); got != "staging" {
		t.Fatalf("Profile() = %q, want %q", got, "staging")
	}
}

func TestManagerUseProfileEmptyAlwaysSucceeds(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(dir, "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "credentials"))

	mgr := NewManager(nil, "", "")
	if err := mgr.UseProfile(context.Background(), ""); err != nil {
		t.Fatalf("UseProfile(\"\"): %v", err)
	}
	if got := mgr.Profile(); got != "" {
		t.Fatalf("Profile() = %q, want empty", got)
	}
}

func TestManagerUseProfileUnknownIsRejected(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config"), "[profile staging]\nregion = us-west-2\n")
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(dir, "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "credentials"))

	mgr := NewManager(nil, "", "")
	if err := mgr.UseProfile(context.Background(), "does-not-exist"); err == nil {
		t.Fatal("expected an error for an unknown profile")
	}
	if got := mgr.Profile(); got != "" {
		t.Fatalf("Profile() = %q, want unchanged empty profile after a rejected switch", got)
	}
}

func TestManagerUseProfileKnownSwitchesActiveProfile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config"), "[profile staging]\nregion = us-west-2\n")
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(dir, "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "credentials"))

	mgr := NewManager(nil, "", "")
	if err := mgr.UseProfile(context.Background(), "staging"); err != nil {
		t.Fatalf("UseProfile: %v", err)
	}
	if got := mgr.Profile(); got != "staging" {
		t.Fatalf("Profile() = %q, want %q", got, "staging")
	}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
