package aws

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/app-sre/go-qontract-reconcile/pkg/vault"
	"github.com/stretchr/testify/assert"
)

func TestGetCredentialsFromEnv(t *testing.T) {
	assert.Nil(t, getCredentialsFromEnv())

	os.Setenv("AWS_ACCESS_KEY_ID", "foo")
	assert.Nil(t, getCredentialsFromEnv())

	os.Setenv("AWS_SECRET_ACCESS_KEY", "bar")
	c := getCredentialsFromEnv()
	assert.NotNil(t, c)
	assert.IsType(t, &Credentials{}, c)
	assert.Equal(t, "foo", c.AccessKeyID)
	assert.Equal(t, "bar", c.SecretAccessKey)
}

func setupVaultMock(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() == "/v1/token" {
			fmt.Fprintf(w, `{"Data": {"aws_access_key_id":"foo", "aws_secret_access_key": "bar"}}`)
		}
	}))
}

func TestGetCredentialsFromVault(t *testing.T) {
	ctx := context.Background()
	toManyAccounts := getAccountsResponse{[]getAccountsAwsaccounts_v1AWSAccount_v1{{}, {}}}

	c, e := getCredentialsFromVault(ctx, nil, &toManyAccounts)
	assert.Nil(t, c)
	assert.NotNil(t, e)

	vaultMock := setupVaultMock(t)

	accounts := getAccountsResponse{
		[]getAccountsAwsaccounts_v1AWSAccount_v1{
			{AutomationToken: getAccountsAwsaccounts_v1AWSAccount_v1AutomationTokenVaultSecret_v1{Path: "token"}},
		},
	}

	os.Setenv("VAULT_TOKEN", "token")
	os.Setenv("VAULT_AUTHTYPE", "token")
	os.Setenv("VAULT_SERVER", vaultMock.URL)
	v, err := vault.NewVaultClient()

	assert.NoError(t, err)
	c, e = getCredentialsFromVault(ctx, v, &accounts)
	assert.NotNil(t, c)
	assert.Nil(t, e)

	assert.Equal(t, "foo", c.AccessKeyID)
	assert.Equal(t, "bar", c.SecretAccessKey)
}

func TestGuessAccountName(t *testing.T) {
	assert.Equal(t, "", guessAccountName())

	os.Setenv("APP_INTERFACE_STATE_BUCKET_ACCOUNT", "foo")
	assert.Equal(t, "foo", guessAccountName())
}