// SPDX-License-Identifier: MIT

package prompts

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rangertaha/aws-mcp/internal/server"
)

func TestRegisterAddsSurveyBucketPrompt(t *testing.T) {
	s := server.New("test", "0.0.0", false)
	Register(s)

	if s.PromptCount() != 1 {
		t.Fatalf("PromptCount() = %d, want 1", s.PromptCount())
	}
}

// TestSurveyBucketRenderSubstitutesBucketName drives the registered prompt
// through a real in-memory MCP client/server round-trip and checks the
// rendered instructions actually contain the caller-supplied bucket name —
// both in the narrative sentence and the example aws_invoke JSON input — and
// no leftover "%s" from an under-substituted format string.
func TestSurveyBucketRenderSubstitutesBucketName(t *testing.T) {
	s := server.New("test", "0.0.0", false)
	Register(s)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	serverErrCh := make(chan error, 1)
	go func() { serverErrCh <- s.Run(ctx, serverTransport) }()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	defer func() { _ = session.Close() }()

	const bucket = "my-test-bucket"
	res, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name:      "survey_bucket",
		Arguments: map[string]string{"bucket": bucket},
	})
	if err != nil {
		t.Fatalf("GetPrompt(survey_bucket): %v", err)
	}
	if len(res.Messages) != 1 {
		t.Fatalf("GetPrompt(survey_bucket) returned %d messages, want 1", len(res.Messages))
	}
	text, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("prompt message content type = %T, want *mcp.TextContent", res.Messages[0].Content)
	}

	if count := strings.Count(text.Text, bucket); count != 2 {
		t.Errorf("rendered prompt mentions %q %d times, want 2 (narrative + JSON input): %s", bucket, count, text.Text)
	}
	if strings.Contains(text.Text, "%s") || strings.Contains(text.Text, "%!s") {
		t.Errorf("rendered prompt still contains an unsubstituted format verb: %s", text.Text)
	}

	cancel()
	<-serverErrCh
}
