// SPDX-License-Identifier: MIT

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Identity describes the calling AWS principal.
type Identity struct {
	Account string
	Arn     string
	UserID  string
}

// Check verifies credentials by calling STS GetCallerIdentity, returning the
// resolved principal.
func Check(ctx context.Context, c *Clients) (*Identity, error) {
	out, err := c.STS.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
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
