// SPDX-License-Identifier: MIT

// Package app assembles the fully-configured aws-mcp server from configuration.
// It is shared by the command entry point (cmd/aws) so the exact server the
// binary runs is the one under test.
package app

import (
	"context"
	"log"
	"os"

	"github.com/rangertaha/aws-mcp/internal/aws"
	"github.com/rangertaha/aws-mcp/internal/aws/s3"
	"github.com/rangertaha/aws-mcp/internal/config"
	"github.com/rangertaha/aws-mcp/internal/prompts"
	"github.com/rangertaha/aws-mcp/internal/server"
)

// Assemble builds the fully-configured server (all enabled toolsets and
// prompts) and returns it with a cleanup function. version is reported to
// clients.
func Assemble(cfg *config.Config, version string) (*server.Server, func(), error) {
	clients, err := aws.NewClients(context.Background(), cfg.Region)
	if err != nil {
		return nil, nil, err
	}

	srv := server.New("aws-mcp", version, cfg.ReadOnly)

	for _, ts := range toolsets() {
		if cfg.ToolsetEnabled(ts.Name) {
			ts.Register(srv, clients)
		}
	}

	// Diagnostics go to stderr; stdout is reserved for the MCP protocol.
	log.SetOutput(os.Stderr)

	prompts.Register(srv)

	return srv, func() {}, nil
}

// toolsets returns every toolset registrar, in registration order. New service
// areas are added here.
func toolsets() []server.Toolset {
	return []server.Toolset{
		{Name: s3.Name, Register: s3.Register},
	}
}
