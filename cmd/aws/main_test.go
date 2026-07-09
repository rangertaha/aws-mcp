// SPDX-License-Identifier: MIT

package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// syncBuffer guards a bytes.Buffer with a mutex. cmd.Stderr is written to by
// a goroutine exec.Cmd spawns internally (since it's a plain io.Writer, not
// an *os.File) for as long as the subprocess is alive, while the test's main
// goroutine reads it in error messages before cmd.Wait() has joined that
// goroutine — a bare strings.Builder there would be an unsynchronized
// concurrent read/write, a real (if narrow — only triggered on a failure
// path) data race.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// isolatedAWSEnv builds an environment for the subprocess that strips every
// inherited AWS_* variable, points the shared config/credentials files at
// paths that don't exist, and disables EC2 IMDS. This test only calls tools
// that need no credentials (aws_list_services, the survey_bucket prompt),
// but the subprocess otherwise inherits the *host's* real AWS environment —
// whatever profile, static keys, or IMDS route happens to be configured
// wherever this test runs — so isolating it defends against any future
// change to this test (or a bug in the tools it calls) accidentally
// resolving real credentials or reaching the network, which this repo's
// test suite must never do.
func isolatedAWSEnv(t *testing.T) []string {
	t.Helper()
	dir := t.TempDir()

	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "AWS_") {
			env = append(env, kv)
		}
	}
	return append(env,
		"AWS_READONLY=true",
		"AWS_EC2_METADATA_DISABLED=true",
		"AWS_CONFIG_FILE="+filepath.Join(dir, "no-such-config"),
		"AWS_SHARED_CREDENTIALS_FILE="+filepath.Join(dir, "no-such-credentials"),
	)
}

// TestMCPStdioIntegration is the only test in this repo that exercises the
// real, compiled binary over its actual stdin/stdout — every other test
// (including internal/server's) drives the server through
// mcp.NewInMemoryTransports(), which bypasses cmd/aws/main.go entirely:
// its .env loading, config.Load(), the urfave/cli "mcp" subcommand
// dispatch, signal.NotifyContext wiring, and real line-delimited JSON-RPC
// framing over OS pipes. aws_list_services needs no AWS credentials or
// network access, so this stays hermetic like the rest of the suite.
func TestMCPStdioIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stdio integration test in -short mode (builds the full binary)")
	}

	bin := filepath.Join(t.TempDir(), "aws-mcp-test-bin")
	build := exec.Command("go", "build", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, "mcp")
	cmd.Env = isolatedAWSEnv(t)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("StdinPipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}
	var stderr syncBuffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("starting %s mcp: %v", bin, err)
	}
	defer func() {
		_ = stdin.Close()
		_ = cmd.Wait()
	}()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)

	send := func(v any) {
		t.Helper()
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshaling request: %v", err)
		}
		if _, err := stdin.Write(append(raw, '\n')); err != nil {
			t.Fatalf("writing to stdin: %v (stderr so far: %s)", err, stderr.String())
		}
	}
	recv := func() map[string]any {
		t.Helper()
		if !scanner.Scan() {
			t.Fatalf("no response on stdout: %v (stderr: %s)", scanner.Err(), stderr.String())
		}
		var v map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &v); err != nil {
			t.Fatalf("unmarshaling response %q: %v", scanner.Text(), err)
		}
		return v
	}

	send(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "integration-test", "version": "0.0.0"},
		},
	})
	initResp := recv()
	if initResp["error"] != nil {
		t.Fatalf("initialize returned an error: %v", initResp["error"])
	}

	send(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})

	send(map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params":  map[string]any{"name": "aws_list_services", "arguments": map[string]any{}},
	})
	callResp := recv()
	if callResp["error"] != nil {
		t.Fatalf("tools/call aws_list_services returned a protocol error: %v", callResp["error"])
	}
	result, ok := callResp["result"].(map[string]any)
	if !ok {
		t.Fatalf("tools/call result = %v, want an object", callResp["result"])
	}
	if isErr, _ := result["isError"].(bool); isErr {
		t.Fatalf("aws_list_services returned a tool error: %v", result["content"])
	}
	structured, ok := result["structuredContent"].(map[string]any)
	if !ok {
		t.Fatalf("result has no structuredContent: %v", result)
	}
	count, _ := structured["count"].(float64)
	if count < 400 {
		t.Errorf("aws_list_services reported %v services over real stdio, want at least 400", structured["count"])
	}

	// Also exercise the prompts path (survey_bucket) over the same real
	// stdio connection, for broader coverage of the actual entrypoint.
	// Deliberately NOT calling aws_whoami or any other credential-touching
	// tool here: this suite must never attempt AWS credential resolution or
	// network access, even in a "designed to fail fast" form.
	send(map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "prompts/get",
		"params":  map[string]any{"name": "survey_bucket", "arguments": map[string]any{"bucket": "my-test-bucket"}},
	})
	promptResp := recv()
	if promptResp["error"] != nil {
		t.Fatalf("prompts/get survey_bucket returned an error: %v", promptResp["error"])
	}
	promptRaw, err := json.Marshal(promptResp["result"])
	if err != nil {
		t.Fatalf("marshaling prompt result: %v", err)
	}
	if !strings.Contains(string(promptRaw), "my-test-bucket") {
		t.Errorf("survey_bucket prompt result over real stdio doesn't mention the bucket name: %s", promptRaw)
	}
}
