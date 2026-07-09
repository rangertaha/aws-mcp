// SPDX-License-Identifier: MIT

package config

import (
	"reflect"
	"strings"
	"testing"
)

func TestSplitList(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"whitespace only", "   ", nil},
		{"single", "s3", []string{"s3"}},
		{"multiple", "s3,ec2,lambda", []string{"s3", "ec2", "lambda"}},
		{"whitespace around entries", " s3 , ec2 ,  lambda ", []string{"s3", "ec2", "lambda"}},
		{"mixed case lower-cased", "S3,Ec2,LAMBDA", []string{"s3", "ec2", "lambda"}},
		{"empty entries dropped", "s3,,ec2,", []string{"s3", "ec2"}},
		{"garbage only commas", ",,,", []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := splitList(c.in)
			if c.want == nil {
				if got != nil {
					t.Errorf("splitList(%q) = %#v, want nil", c.in, got)
				}
				return
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("splitList(%q) = %#v, want %#v", c.in, got, c.want)
			}
		})
	}
}

// FuzzSplitList checks splitList never panics on arbitrary input and that
// its documented invariants always hold: every returned entry is non-empty,
// trimmed, and lower-cased.
func FuzzSplitList(f *testing.F) {
	for _, seed := range []string{
		"", ",", ",,,", "s3,ec2", " s3 , ec2 ", "S3,EC2", "s3,,ec2,",
		"\t\n", "s3\x00ec2", "😀,s3", strings.Repeat("a,", 1000),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, in string) {
		got := splitList(in)
		for _, entry := range got {
			if entry == "" {
				t.Fatalf("splitList(%q) contains an empty entry: %#v", in, got)
			}
			if entry != strings.TrimSpace(entry) {
				t.Fatalf("splitList(%q) contains an untrimmed entry %q: %#v", in, entry, got)
			}
			if entry != strings.ToLower(entry) {
				t.Fatalf("splitList(%q) contains a non-lower-cased entry %q: %#v", in, entry, got)
			}
		}
	})
}

// FuzzIsTruthy checks isTruthy never panics and stays consistent regardless
// of surrounding whitespace or case, since its doc comment promises both are
// ignored.
func FuzzIsTruthy(f *testing.F) {
	for _, seed := range []string{
		"", "true", "TRUE", " true ", "1", "0", "false", "yes", "no", "😀", "\x00",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, in string) {
		got := isTruthy(in)
		if wrapped := isTruthy("  " + in + "  "); wrapped != got {
			t.Errorf("isTruthy(%q)=%v but isTruthy(%q)=%v: whitespace should be ignored", in, got, "  "+in+"  ", wrapped)
		}
		if upper := isTruthy(strings.ToUpper(in)); upper != got {
			t.Errorf("isTruthy(%q)=%v but isTruthy(%q)=%v: case should be ignored", in, got, strings.ToUpper(in), upper)
		}
	})
}

func TestIsTruthy(t *testing.T) {
	truthy := []string{"1", "true", "True", "TRUE", "yes", "Yes", "on", "On", " true ", "\ttrue\n"}
	for _, v := range truthy {
		if !isTruthy(v) {
			t.Errorf("isTruthy(%q) = false, want true", v)
		}
	}
	falsy := []string{"", "0", "false", "False", "no", "off", "2", "yesplease", " ", "null"}
	for _, v := range falsy {
		if isTruthy(v) {
			t.Errorf("isTruthy(%q) = true, want false", v)
		}
	}
}

func TestConfigAllToolsets(t *testing.T) {
	cases := []struct {
		name     string
		toolsets []string
		want     bool
	}{
		{"nil", nil, true},
		{"empty slice", []string{}, true},
		{"specific toolsets", []string{"s3", "ec2"}, false},
		{"explicit all", []string{"all"}, true},
		{"all mixed with others", []string{"s3", "all"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := &Config{Toolsets: c.toolsets}
			if got := cfg.AllToolsets(); got != c.want {
				t.Errorf("AllToolsets() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestConfigToolsetEnabled(t *testing.T) {
	all := &Config{}
	if !all.ToolsetEnabled("s3") {
		t.Error("empty Toolsets should enable every toolset")
	}

	specific := &Config{Toolsets: []string{"s3", "ec2"}}
	if !specific.ToolsetEnabled("s3") {
		t.Error("s3 should be enabled")
	}
	if !specific.ToolsetEnabled("S3") {
		t.Error("ToolsetEnabled should be case-insensitive")
	}
	if specific.ToolsetEnabled("lambda") {
		t.Error("lambda should not be enabled")
	}
}

func TestLoad(t *testing.T) {
	t.Setenv(EnvRegion, "  us-west-2  ")
	t.Setenv(EnvToolsets, "S3, EC2")
	t.Setenv(EnvReadOnly, "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Region != "us-west-2" {
		t.Errorf("Region = %q, want trimmed %q", cfg.Region, "us-west-2")
	}
	if !reflect.DeepEqual(cfg.Toolsets, []string{"s3", "ec2"}) {
		t.Errorf("Toolsets = %#v, want [s3 ec2]", cfg.Toolsets)
	}
	if !cfg.ReadOnly {
		t.Error("ReadOnly = false, want true")
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv(EnvRegion, "")
	t.Setenv(EnvToolsets, "")
	t.Setenv(EnvReadOnly, "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Region != "" {
		t.Errorf("Region = %q, want empty", cfg.Region)
	}
	if !cfg.AllToolsets() {
		t.Error("unset AWS_TOOLSETS should mean all toolsets")
	}
	if cfg.ReadOnly {
		t.Error("unset AWS_READONLY should mean ReadOnly=false")
	}
}
