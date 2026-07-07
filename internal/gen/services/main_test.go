// SPDX-License-Identifier: MIT

package main

import (
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

func TestLoadEntriesRejectsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "services.json")
	writeFile(t, path, `not json`)

	if _, err := loadEntries(path); err == nil {
		t.Fatal("expected an error for invalid JSON")
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
