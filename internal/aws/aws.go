// SPDX-License-Identifier: MIT

// Package aws holds the AWS SDK clients that the per-service tool packages
// (s3, …) share. Unlike the other servers in this family, aws-mcp uses the
// official aws-sdk-go-v2 rather than a hand-rolled REST client, so credentials
// come from the standard AWS credential chain (environment, shared config,
// SSO, or an attached IAM role).
package aws

import (
	"context"
	"fmt"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Clients bundles the AWS service clients used by the toolsets.
type Clients struct {
	// Region is the resolved AWS region.
	Region string
	// S3 reaches Amazon S3.
	S3 *s3.Client
	// STS reaches AWS Security Token Service (used by the connectivity check).
	STS *sts.Client
}

// NewClients loads AWS configuration from the default credential chain and
// builds the service clients. An explicit region overrides the chain-resolved
// one when non-empty.
func NewClients(ctx context.Context, region string) (*Clients, error) {
	opts := []func(*awsconfig.LoadOptions) error{}
	if region != "" {
		opts = append(opts, awsconfig.WithRegion(region))
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}
	return &Clients{
		Region: cfg.Region,
		S3:     s3.NewFromConfig(cfg),
		STS:    sts.NewFromConfig(cfg),
	}, nil
}
