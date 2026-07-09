// SPDX-License-Identifier: MIT

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadEntriesSortsByName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "services.json")
	writeFile(t, path, `{
		"s3": "github.com/aws/aws-sdk-go-v2/service/s3",
		"ec2": "github.com/aws/aws-sdk-go-v2/service/ec2"
	}`)

	entries, err := loadEntries(path)
	if err != nil {
		t.Fatalf("loadEntries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("loadEntries() returned %d entries, want 2", len(entries))
	}
	if entries[0].Name != "ec2" || entries[1].Name != "s3" {
		t.Fatalf("loadEntries() not sorted by name: %+v", entries)
	}
	if entries[0].Alias != "ec2" || entries[0].Import != "github.com/aws/aws-sdk-go-v2/service/ec2" {
		t.Fatalf("unexpected entry: %+v", entries[0])
	}
}

func TestLoadEntriesRejectsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "services.json")
	writeFile(t, path, `{}`)

	if _, err := loadEntries(path); err == nil {
		t.Fatal("expected an error for a services.json with no entries")
	}
}

func TestLoadEntriesRejectsMissingFile(t *testing.T) {
	if _, err := loadEntries("/no/such/path/services.json"); err == nil {
		t.Fatal("expected an error for a missing file")
	}
}

func TestLoadEntriesRejectsInvalidGoIdentifierName(t *testing.T) {
	cases := []string{
		"s3-legacy",  // hyphen: not a valid identifier
		"3dsecure",   // leading digit
		"my service", // space
		"",           // empty key (valid JSON, invalid identifier)
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "services.json")
			body, err := json.Marshal(map[string]string{name: "github.com/aws/aws-sdk-go-v2/service/s3"})
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, path, string(body))

			_, err = loadEntries(path)
			if err == nil {
				t.Fatalf("loadEntries with service name %q: expected an error, got none", name)
			}
			if !strings.Contains(err.Error(), "valid Go identifier") {
				t.Errorf("error = %q, want it to explain the identifier requirement", err.Error())
			}
		})
	}
}

func TestLoadEntriesRejectsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "services.json")
	writeFile(t, path, `not json`)

	if _, err := loadEntries(path); err == nil {
		t.Fatal("expected an error for invalid JSON")
	}
}

// TestRealServicesJSONLoads guards against a future hand-edit of the real
// services.json introducing a name that isn't a valid Go identifier — the
// exact scenario TestLoadEntriesRejectsInvalidGoIdentifierName exercises
// synthetically, checked here against the actual committed file.
func TestRealServicesJSONLoads(t *testing.T) {
	// Tests run with the package directory as cwd, unlike run()'s
	// defaultServicesPath (which is relative to the repo root for `go run
	// ./internal/gen/services`) — the file itself lives right alongside
	// this test.
	const path = "services.json"
	entries, err := loadEntries(path)
	if err != nil {
		t.Fatalf("loadEntries(%s): %v", path, err)
	}
	if len(entries) < 400 {
		t.Fatalf("loadEntries(%s) = %d entries, want at least 400", path, len(entries))
	}
}

func TestRenderProducesFormattedGoSource(t *testing.T) {
	entries := []entry{
		{Name: "ec2", Import: "github.com/aws/aws-sdk-go-v2/service/ec2", Alias: "ec2"},
		{Name: "s3", Import: "github.com/aws/aws-sdk-go-v2/service/s3", Alias: "s3"},
	}

	src, err := render(entries)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	got := string(src)
	for _, want := range []string{
		"package registry",
		`ec2 "github.com/aws/aws-sdk-go-v2/service/ec2"`,
		`s3 "github.com/aws/aws-sdk-go-v2/service/s3"`,
		`"ec2": func(cfg awssdk.Config) any { return ec2.NewFromConfig(cfg) }`,
		`func(cfg awssdk.Config) any { return s3.NewFromConfig(cfg) }`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("render() output missing %q; got:\n%s", want, got)
		}
	}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
