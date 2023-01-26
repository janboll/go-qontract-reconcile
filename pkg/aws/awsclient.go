// inspired by https://github.com/openshift/aws-account-operator/blob/master/pkg/awsclient/client.go

package aws

import (
	"context"

	"github.com/app-sre/go-qontract-reconcile/pkg/util"
	"github.com/app-sre/go-qontract-reconcile/pkg/vault"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/viper"
)

//go:generate go run github.com/Khan/genqlient

var _ = `# @genqlient
query getAccounts($name: String) {
	awsaccounts_v1 (name: $name) {
		name
		resourcesDefaultRegion
		automationToken {
			path
			field
			version
			format
		}
	}
}
`

//go:generate mockgen -source=./awsclient.go -destination=./mock/zz_generated.mock_client.go -package=mock

// Client is a wrapper object for actual AWS SDK clients to allow for easier testing.
type Client interface {
	//S3
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

type awsClient struct {
	s3Client s3.Client
}

func (c *awsClient) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return c.s3Client.GetObject(ctx, params, optFns...)
}

func (c *awsClient) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return c.s3Client.HeadObject(ctx, params, optFns...)
}

func (c *awsClient) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return c.s3Client.PutObject(ctx, params, optFns...)
}

func (c *awsClient) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return c.s3Client.DeleteObject(ctx, params, optFns...)
}

type awsClientConfig struct {
	Region string
}

func newAwsClientConfig() *awsClientConfig {
	var cfg awsClientConfig
	sub := util.EnsureViperSub(viper.GetViper(), "aws")
	sub.BindEnv("region", "AWS_REGION")
	if err := sub.Unmarshal(&cfg); err != nil {
		util.Log().Fatalw("Error while unmarshalling configuration %s", err.Error())
	}
	return &cfg
}

func NewClient(ctx context.Context, vc vault.VaultClient, account string) *awsClient {
	if len(account) == 0 {
		util.Log().Fatalw("No AWS account name provided")
	}
	result, err := getAccounts(ctx, account)
	if err != nil {
		util.Log().Fatalw("Error getting AWS account info", "error", err.Error())
	}
	accounts := result.GetAwsaccounts_v1()
	if len(accounts) != 1 {
		util.Log().Fatalw("Expected one AWS with name", "account", account)

	}

	secret, err := vc.ReadSecret(accounts[0].AutomationToken.GetPath())
	if err != nil {
		util.Log().Fatalw("Error reading automation token", "error", err.Error())
	}
	awsCfg := newAwsClientConfig()

	aws_access_key_id := secret.Data["aws_access_key_id"].(string)
	aws_secret_access_key := secret.Data["aws_secret_access_key"].(string)
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(awsCfg.Region), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(aws_access_key_id, aws_secret_access_key, "")))
	if err != nil {
		util.Log().Fatalw("Error creating AWS configuration", "error", err.Error())
	}

	return &awsClient{
		s3Client: *s3.NewFromConfig(cfg),
	}
}