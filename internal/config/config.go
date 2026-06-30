// SPDX-License-Identifier: MIT

// Package config loads and validates runtime configuration for the aws-mcp
// server from environment variables.
//
// Credentials are NOT read here: aws-mcp uses the standard AWS credential chain
// (environment, shared config/credentials files, SSO, or an attached IAM role)
// via aws-sdk-go-v2. This package only carries server behavior plus an optional
// region override.
package config

import (
	"os"
	"strings"
)

// Environment variable names recognised by the server. AWS_REGION is the
// standard SDK variable and is reused here as the region override.
const (
	EnvRegion   = "AWS_REGION"   // optional region override (standard AWS var)
	EnvToolsets = "AWS_TOOLSETS" // comma-separated toolset names, or "all"
	EnvReadOnly = "AWS_READONLY" // "true" disables all write tools
)

// Config holds validated server configuration.
type Config struct {
	// Region overrides the credential-chain region when non-empty.
	Region string
	// Toolsets is the set of enabled toolset names. A nil/empty set means "all".
	Toolsets []string
	// ReadOnly, when true, suppresses mutating tools at registration time.
	ReadOnly bool
}

// AllToolsets reports whether every toolset should be enabled.
func (c *Config) AllToolsets() bool {
	if len(c.Toolsets) == 0 {
		return true
	}
	for _, t := range c.Toolsets {
		if t == "all" {
			return true
		}
	}
	return false
}

// ToolsetEnabled reports whether the named toolset should be registered.
func (c *Config) ToolsetEnabled(name string) bool {
	if c.AllToolsets() {
		return true
	}
	for _, t := range c.Toolsets {
		if strings.EqualFold(t, name) {
			return true
		}
	}
	return false
}

// Load reads configuration from the process environment. aws-mcp has no
// required configuration (credentials come from the AWS chain), so Load never
// fails today; it returns an error for signature parity with the other servers.
func Load() (*Config, error) {
	return &Config{
		Region:   strings.TrimSpace(os.Getenv(EnvRegion)),
		Toolsets: splitList(os.Getenv(EnvToolsets)),
		ReadOnly: isTruthy(os.Getenv(EnvReadOnly)),
	}, nil
}

// splitList parses a comma-separated environment value into a trimmed,
// lower-cased slice, dropping empty entries.
func splitList(v string) []string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.ToLower(strings.TrimSpace(p)); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// isTruthy reports whether an environment value represents boolean true.
func isTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
