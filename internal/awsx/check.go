// SPDX-License-Identifier: MIT

package awsx

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Identity describes the calling AWS principal.
type Identity struct {
	Account string `json:"account" jsonschema:"AWS account ID"`
	Arn     string `json:"arn" jsonschema:"ARN of the calling principal"`
	UserID  string `json:"userId" jsonschema:"unique identifier of the calling principal"`
}

// Check verifies credentials by calling STS GetCallerIdentity against the
// manager's currently active profile, returning the resolved principal. STS
// is called directly here (not through the generic registry/dispatch) since
// this is an internal connectivity check, not a model-facing operation.
func Check(ctx context.Context, m *Manager) (*Identity, error) {
	cfg, err := m.Config(ctx)
	if err != nil {
		return nil, err
	}

	out, err := sts.NewFromConfig(cfg).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	}

	id := &Identity{}
	if out.Account != nil {
		id.Account = *out.Account
	}
	if out.Arn != nil {
		id.Arn = *out.Arn
	}
	if out.UserId != nil {
		id.UserID = *out.UserId
	}
	return id, nil
}
