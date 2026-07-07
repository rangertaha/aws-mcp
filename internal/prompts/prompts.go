// SPDX-License-Identifier: MIT

// Package prompts registers MCP prompts: user-invoked, parameterized templates
// that clients surface as slash commands. Each prompt encodes a multi-step
// workflow by guiding the model to call the right tools in order.
package prompts

import (
	"fmt"

	"github.com/rangertaha/aws-mcp/internal/server"
)

// Register adds the built-in workflow prompts to the server.
func Register(s *server.Server) {
	s.AddPrompt(
		"survey_bucket",
		"Survey an S3 bucket: list its top-level contents and summarize size and object count.",
		[]server.PromptArg{
			{Name: "bucket", Description: "S3 bucket name", Required: true},
		},
		func(a map[string]string) string {
			return fmt.Sprintf(`Survey the S3 bucket "%s".

Steps:
1. Call aws_invoke (service="s3", operation="ListObjectsV2", input={"Bucket": "%s"}) to list its objects.
2. Report the number of objects returned and the total size.
3. Group keys by their top-level prefix and summarize what the bucket appears to hold.`,
				a["bucket"], a["bucket"])
		},
	)
}
