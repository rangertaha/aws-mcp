// SPDX-License-Identifier: MIT

// Command aws runs the AWS Model Context Protocol server (`aws mcp`) and checks
// connectivity (`aws test`).
//
// Credentials come from the standard AWS credential chain (environment, shared
// config, SSO, or an attached IAM role). The `mcp` command communicates over
// stdio, the transport expected by MCP clients such as Claude Desktop/Code and
// Cursor.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/urfave/cli/v3"

	"github.com/rangertaha/aws-mcp/internal"
	"github.com/rangertaha/aws-mcp/internal/app"
	"github.com/rangertaha/aws-mcp/internal/awsx"
	"github.com/rangertaha/aws-mcp/internal/awsx/registry"
	"github.com/rangertaha/aws-mcp/internal/config"
)

func main() {
	cmd := &cli.Command{
		Name:    "aws",
		Usage:   "AWS services as an MCP server",
		Version: internal.Version(),
		// A bare `aws` (no subcommand) runs the MCP server.
		Action: runMCP,
		Commands: []*cli.Command{
			mcpCommand(),
			testCommand(),
		},
		// Print errors ourselves so the MCP stdio stream is never touched.
		ExitErrHandler: func(context.Context, *cli.Command, error) {},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "aws: %v\n", err)
		os.Exit(1)
	}
}

// mcpCommand runs the MCP server over stdio.
func mcpCommand() *cli.Command {
	return &cli.Command{
		Name:   "mcp",
		Usage:  "Run the MCP server over stdio (for Claude Desktop/Code, Cursor, ...)",
		Action: runMCP,
	}
}

// runMCP assembles and serves the MCP server over stdio.
func runMCP(ctx context.Context, _ *cli.Command) error {
	if err := config.LoadEnvFile(config.EnvFile); err != nil {
		log.Printf("aws: reading %s: %v", config.EnvFile, err)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("configuration error:\n%w", err)
	}

	ver := internal.Version()
	srv, cleanup, err := app.Assemble(cfg, ver)
	if err != nil {
		return err
	}
	defer cleanup()

	log.Printf("aws-mcp %s starting: %d tools, %d prompts across toolsets %v (read-only=%v)",
		ver, srv.ToolCount(), srv.PromptCount(), srv.Toolsets(), cfg.ReadOnly)

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return srv.Run(ctx, &mcp.StdioTransport{})
}

// testCommand verifies credentials via STS GetCallerIdentity.
func testCommand() *cli.Command {
	return &cli.Command{
		Name:  "test",
		Usage: "Test AWS credentials (STS GetCallerIdentity)",
		Action: func(ctx context.Context, _ *cli.Command) error {
			if err := config.LoadEnvFile(config.EnvFile); err != nil {
				log.Printf("aws: reading %s: %v", config.EnvFile, err)
			}

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("configuration error:\n%w", err)
			}

			ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			mgr := awsx.NewManager(registry.Factories, cfg.Region, "")

			id, err := awsx.Check(ctx, mgr)
			if err != nil {
				return fmt.Errorf("verifying AWS credentials: %w", err)
			}

			sdkCfg, err := mgr.Config(ctx)
			if err != nil {
				return err
			}

			fmt.Printf("OK  authenticated with AWS (region=%s)\n", sdkCfg.Region)
			fmt.Printf("    account=%s arn=%s\n", id.Account, id.Arn)
			fmt.Printf("    read-only=%v\n", cfg.ReadOnly)
			return nil
		},
	}
}
