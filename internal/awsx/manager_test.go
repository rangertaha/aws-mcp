// SPDX-License-Identifier: MIT

package awsx

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"

	"github.com/rangertaha/aws-mcp/internal/awsx/registry"
)

type fakeClient struct{ id int64 }

// countingFactory's counter is an atomic.Int64, not a plain int: it's shared
// across every call the factory makes, and TestManagerConcurrentUseProfileAndClientStress
// calls it from many goroutines at once (Manager.Client doesn't serialize
// concurrent factory invocations on a cache miss — see the comment on
// Manager.Client — so a factory reachable from concurrent callers must be
// safe to call concurrently itself).
func countingFactory(calls *atomic.Int64) registry.ClientFactory {
	return func(awssdk.Config) any {
		return &fakeClient{id: calls.Add(1)}
	}
}

func TestManagerClientCachesPerProfile(t *testing.T) {
	var calls atomic.Int64
	mgr := NewManager(map[string]registry.ClientFactory{"fake": countingFactory(&calls)}, "", "")

	first, err := mgr.Client(context.Background(), "fake")
	if err != nil {
		t.Fatalf("Client: %v", err)
	}
	second, err := mgr.Client(context.Background(), "fake")
	if err != nil {
		t.Fatalf("Client: %v", err)
	}

	if got := calls.Load(); got != 1 {
		t.Fatalf("factory called %d times, want 1", got)
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

// TestManagerUseProfileSucceedsWithBogusCredentials pins down documented
// (not buggy) behavior: UseProfile validates that the profile name is known
// and that its static config resolves, but does not verify the resulting
// credentials actually work. A profile with well-formed but fake static
// keys switches successfully — the failure is deferred to the first real
// AWS call, which is the SDK's own credential-chain design, not something
// UseProfile can or should short-circuit without an extra network round
// trip on every profile switch.
func TestManagerUseProfileSucceedsWithBogusCredentials(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config"), "[profile bogus]\nregion = us-east-1\n")
	writeFile(t, filepath.Join(dir, "credentials"),
		"[bogus]\naws_access_key_id = AKIAFAKEFAKEFAKEFAKE\naws_secret_access_key = fakefakefakefakefakefakefakefakefakefake\n")
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(dir, "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "credentials"))

	mgr := NewManager(nil, "", "")
	if err := mgr.UseProfile(context.Background(), "bogus"); err != nil {
		t.Fatalf("UseProfile with well-formed but fake credentials should succeed (validity is checked on first use, not here): %v", err)
	}
	if got := mgr.Profile(); got != "bogus" {
		t.Fatalf("Profile() = %q, want %q", got, "bogus")
	}
}

// TestManagerClientProfileSwitchRace pins down a concurrency bug: Client
// used to snapshot the active profile once for its cache key, then re-read
// the (possibly now different) active profile a second time inside Config
// to resolve the aws.Config used to actually build the client — so a
// UseProfile call landing in that window built a client against the NEW
// profile's config but cached it under the OLD profile's key: a subsequent
// call under the old profile would silently reuse a client holding the new
// profile's credentials. Uses the afterProfileSnapshot hook (not sleeps) to
// deterministically land UseProfile in that exact window, which has no
// naturally hookable I/O of its own to synchronize on (it's a
// same-goroutine, no-I/O gap between two lock/read pairs).
func TestManagerClientProfileSwitchRace(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config"), "[default]\nregion = us-east-1\n\n[profile other]\nregion = us-west-2\n")
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(dir, "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "credentials"))

	origHook := afterProfileSnapshot
	t.Cleanup(func() { afterProfileSnapshot = origHook })

	snapshotted := make(chan struct{})
	proceed := make(chan struct{})
	afterProfileSnapshot = func() {
		close(snapshotted)
		<-proceed
	}

	// The factory records which config it was actually built with, via the
	// distinguishing region — this is the crux of the test: the bug doesn't
	// change *where* the client is cached (the cache-write key is always
	// the originally snapshotted "profile" local variable, in both the
	// buggy and fixed code), it changes *what config the client was built
	// from*.
	var builtWithRegion string
	factory := func(cfg awssdk.Config) any {
		builtWithRegion = cfg.Region
		return &fakeClient{}
	}
	mgr := NewManager(map[string]registry.ClientFactory{"fake": factory}, "", "")

	type result struct {
		client any
		err    error
	}
	done := make(chan result, 1)
	go func() {
		c, err := mgr.Client(context.Background(), "fake")
		done <- result{c, err}
	}()

	<-snapshotted // Client() has snapshotted profile="" (default) and is about to resolve its config.
	if err := mgr.UseProfile(context.Background(), "other"); err != nil {
		t.Fatalf("UseProfile: %v", err)
	}
	close(proceed) // let Client() proceed to resolve its config and return.

	res := <-done
	if res.err != nil {
		t.Fatalf("Client: %v", res.err)
	}

	if builtWithRegion != "us-east-1" {
		t.Errorf("client built with region %q, want %q (the default profile active when Client() was called, not %q from the profile a concurrent UseProfile switched to mid-call)",
			builtWithRegion, "us-east-1", "other")
	}

	mgr.mu.RLock()
	_, cachedUnderCallTimeProfile := mgr.clients[""]["fake"]
	mgr.mu.RUnlock()
	if !cachedUnderCallTimeProfile {
		t.Error("client not cached under the profile active when Client() was called")
	}
}

// TestManagerConcurrentUseProfileAndClientStress goes beyond
// TestManagerClientProfileSwitchRace's single deterministic window: it
// hammers UseProfile and Client from many goroutines simultaneously (run
// with -race) to catch any unguarded field access that a single targeted
// scenario might miss. It doesn't assert which profile ends up active
// (concurrent switches racing each other is expected, ordinary
// last-writer-wins — not a bug to fix), only that nothing panics, every
// call succeeds or fails cleanly, and the race detector finds no data race.
func TestManagerConcurrentUseProfileAndClientStress(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config"), "[default]\nregion = us-east-1\n\n[profile other]\nregion = us-west-2\n")
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(dir, "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "credentials"))

	var calls atomic.Int64
	mgr := NewManager(map[string]registry.ClientFactory{"fake": countingFactory(&calls)}, "", "")

	const workers = 20
	var wg sync.WaitGroup
	wg.Add(workers * 3)

	for i := 0; i < workers; i++ {
		go func(i int) {
			defer wg.Done()
			profile := ""
			if i%2 == 0 {
				profile = "other"
			}
			if err := mgr.UseProfile(context.Background(), profile); err != nil {
				t.Errorf("UseProfile(%q): %v", profile, err)
			}
		}(i)
		go func() {
			defer wg.Done()
			if _, err := mgr.Client(context.Background(), "fake"); err != nil {
				t.Errorf("Client: %v", err)
			}
		}()
		go func() {
			defer wg.Done()
			_ = mgr.Profile()
		}()
	}
	wg.Wait()
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
