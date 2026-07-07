module github.com/rangertaha/aws-mcp

go 1.25.0

toolchain go1.25.11

require (
	github.com/aws/aws-sdk-go-v2/config v1.32.26
	github.com/aws/aws-sdk-go-v2/service/s3 v1.105.0
	github.com/aws/aws-sdk-go-v2/service/sts v1.44.0
	github.com/google/jsonschema-go v0.4.3
	github.com/modelcontextprotocol/go-sdk v1.6.1
	github.com/urfave/cli/v3 v3.10.1
)

require (
	github.com/aws/aws-sdk-go-v2 v1.42.1 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.14 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.19.25 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.29 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.31 // indirect
	github.com/aws/aws-sdk-go-v2/service/acm v1.42.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/apigateway v1.41.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/apigatewayv2 v1.36.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/appsync v1.55.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/athena v1.59.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/backup v1.58.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/batch v1.67.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/bedrockruntime v1.55.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/cloudformation v1.74.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/cloudfront v1.66.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/cloudtrail v1.57.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.62.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/codebuild v1.71.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/codedeploy v1.37.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/codepipeline v1.48.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider v1.64.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/comprehend v1.42.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/docdb v1.50.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.60.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.312.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecr v1.59.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecs v1.87.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/efs v1.43.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/eks v1.89.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/elasticache v1.55.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk v1.36.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/emr v1.62.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/eventbridge v1.47.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/firehose v1.45.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/glue v1.148.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/iam v1.55.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.12.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.30 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.31 // indirect
	github.com/aws/aws-sdk-go-v2/service/kinesis v1.45.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/kms v1.54.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/lambda v1.95.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/neptune v1.47.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/opensearch v1.74.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/organizations v1.52.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/rds v1.120.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/redshift v1.64.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/rekognition v1.53.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/route53 v1.64.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sagemaker v1.257.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.43.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sfn v1.44.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.2.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sns v1.41.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sqs v1.45.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssm v1.70.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.31.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.36.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/textract v1.42.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/transcribe v1.57.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/wafv2 v1.75.0 // indirect
	github.com/aws/smithy-go v1.27.3 // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	golang.org/x/oauth2 v0.35.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
)
