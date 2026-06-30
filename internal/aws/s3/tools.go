// SPDX-License-Identifier: MIT

package s3

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rangertaha/aws-mcp/internal/aws"
	"github.com/rangertaha/aws-mcp/internal/server"
)

// Register adds the S3 toolset to the server.
func Register(s *server.Server, c *aws.Clients) {
	s.NoteToolset(Name)
	svc := &service{c: c}

	server.Register(s, server.ToolDef{
		Name:        "s3_list_buckets",
		Title:       "List S3 buckets",
		Description: "List the S3 buckets owned by the AWS account.",
	}, svc.listBuckets)

	server.Register(s, server.ToolDef{
		Name:        "s3_list_objects",
		Title:       "List S3 objects",
		Description: "List objects in an S3 bucket, optionally filtered by key prefix.",
	}, svc.listObjects)
}

// EmptyInput is used by tools that take no arguments.
type EmptyInput struct{}

// ListObjectsInput identifies a bucket and optional prefix/limit.
type ListObjectsInput struct {
	Bucket  string `json:"bucket" jsonschema:"S3 bucket name"`
	Prefix  string `json:"prefix,omitempty" jsonschema:"only keys starting with this prefix (optional)"`
	MaxKeys int    `json:"maxKeys,omitempty" jsonschema:"maximum number of objects to return (optional)"`
}

func (s *service) listBuckets(ctx context.Context, _ *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, server.ListResult[Bucket], error) {
	out, err := s.ListBuckets(ctx)
	return nil, server.List(out), err
}

func (s *service) listObjects(ctx context.Context, _ *mcp.CallToolRequest, in ListObjectsInput) (*mcp.CallToolResult, server.ListResult[Object], error) {
	out, err := s.ListObjects(ctx, in.Bucket, in.Prefix, in.MaxKeys)
	return nil, server.List(out), err
}
