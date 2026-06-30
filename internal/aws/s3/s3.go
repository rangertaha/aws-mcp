// SPDX-License-Identifier: MIT

// Package s3 exposes read-only Amazon S3 operations: listing buckets and the
// objects within a bucket.
package s3

import (
	"context"

	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/rangertaha/aws-mcp/internal/aws"
)

// Name is the toolset name used for enable/disable filtering.
const Name = "s3"

// service wraps the AWS clients for S3 operations.
type service struct {
	c *aws.Clients
}

// Bucket is an S3 bucket, trimmed to the fields useful to an LLM.
type Bucket struct {
	Name         string `json:"name"`
	CreationDate string `json:"creationDate,omitempty"`
}

// Object is an S3 object summary.
type Object struct {
	Key          string `json:"key"`
	Size         int64  `json:"size"`
	LastModified string `json:"lastModified,omitempty"`
	StorageClass string `json:"storageClass,omitempty"`
}

// ListBuckets returns the buckets owned by the account.
func (s *service) ListBuckets(ctx context.Context) ([]Bucket, error) {
	out, err := s.c.S3.ListBuckets(ctx, &awss3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}
	buckets := make([]Bucket, 0, len(out.Buckets))
	for _, b := range out.Buckets {
		bk := Bucket{}
		if b.Name != nil {
			bk.Name = *b.Name
		}
		if b.CreationDate != nil {
			bk.CreationDate = b.CreationDate.Format(timeLayout)
		}
		buckets = append(buckets, bk)
	}
	return buckets, nil
}

// ListObjects returns up to maxKeys objects under an optional prefix in a bucket.
func (s *service) ListObjects(ctx context.Context, bucket, prefix string, maxKeys int) ([]Object, error) {
	in := &awss3.ListObjectsV2Input{Bucket: &bucket}
	if prefix != "" {
		in.Prefix = &prefix
	}
	if maxKeys > 0 {
		mk := int32(maxKeys)
		in.MaxKeys = &mk
	}
	out, err := s.c.S3.ListObjectsV2(ctx, in)
	if err != nil {
		return nil, err
	}
	return toObjects(out.Contents), nil
}

// toObjects converts SDK object summaries to the trimmed Object shape.
func toObjects(contents []types.Object) []Object {
	objs := make([]Object, 0, len(contents))
	for _, o := range contents {
		ob := Object{StorageClass: string(o.StorageClass)}
		if o.Key != nil {
			ob.Key = *o.Key
		}
		if o.Size != nil {
			ob.Size = *o.Size
		}
		if o.LastModified != nil {
			ob.LastModified = o.LastModified.Format(timeLayout)
		}
		objs = append(objs, ob)
	}
	return objs
}

// timeLayout is the RFC 3339 layout used for timestamps in tool output.
const timeLayout = "2006-01-02T15:04:05Z07:00"
