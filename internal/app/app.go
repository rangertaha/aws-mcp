// SPDX-License-Identifier: MIT

// Package app assembles the fully-configured aws-mcp server from configuration.
// It is shared by the command entry point (cmd/aws) so the exact server the
// binary runs is the one under test.
package app

import (
	"log"
	"os"

	"github.com/rangertaha/aws-mcp/internal/awsx"
	"github.com/rangertaha/aws-mcp/internal/awsx/registry"
	"github.com/rangertaha/aws-mcp/internal/awsx/tools"
	"github.com/rangertaha/aws-mcp/internal/config"
	"github.com/rangertaha/aws-mcp/internal/prompts"
	"github.com/rangertaha/aws-mcp/internal/server"
)

// Assemble builds the fully-configured server (every enabled AWS service and
// the built-in prompts) and returns it with a cleanup function. version is
// reported to clients.
func Assemble(cfg *config.Config, version string) (*server.Server, func(), error) {
	factories := enabledFactories(cfg)
	mgr := awsx.NewManager(factories, cfg.Region, "")
	cat := registry.Build(factories)

	srv := server.New("aws-mcp", version, cfg.ReadOnly)
	tools.Register(srv, mgr, cat)

	// Diagnostics go to stderr; stdout is reserved for the MCP protocol.
	log.SetOutput(os.Stderr)

	prompts.Register(srv)

	return srv, func() {}, nil
}

// enabledFactories filters registry.Factories down to the services enabled by
// cfg's AWS_TOOLSETS setting (a nil/"all" setting keeps every service).
func enabledFactories(cfg *config.Config) map[string]registry.ClientFactory {
	if cfg.AllToolsets() {
		return registry.Factories
	}
	out := make(map[string]registry.ClientFactory)
	for name, f := range registry.Factories {
		if cfg.ToolsetEnabled(name) {
			out[name] = f
		}
	}
	return out
}
