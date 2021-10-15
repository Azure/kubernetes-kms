// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"

	"github.com/Azure/kubernetes-kms/pkg/auth"
	"github.com/Azure/kubernetes-kms/pkg/config"
	"github.com/Azure/kubernetes-kms/pkg/utils"
	"github.com/Azure/kubernetes-kms/pkg/version"

	kv "github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest/azure"
	"k8s.io/klog/v2"
)

// Client interface for interacting with Keyvault
type Client interface {
	Encrypt(ctx context.Context, cipher []byte) ([]byte, error)
	Decrypt(ctx context.Context, plain []byte) ([]byte, error)
}

type keyVaultClient struct {
	baseClient       kv.BaseClient
	config           *config.AzureConfig
	vaultName        string
	keyName          string
	keyVersion       string
	vaultURL         string
	azureEnvironment *azure.Environment
}

// NewKeyVaultClient returns a new key vault client to use for kms operations
func newKeyVaultClient(config *config.AzureConfig, vaultName, keyName, keyVersion string, proxyMode bool, proxyAddress string, proxyPort int) (*keyVaultClient, error) {
	// Sanitize vaultName, keyName, keyVersion. (https://github.com/Azure/kubernetes-kms/issues/85)
	vaultName = utils.SanitizeString(vaultName)
	keyName = utils.SanitizeString(keyName)
	keyVersion = utils.SanitizeString(keyVersion)

	// this should be the case for bring your own key, clusters bootstrapped with
	// aks-engine or aks and standalone kms plugin deployments
	if len(vaultName) == 0 || len(keyName) == 0 || len(keyVersion) == 0 {
		return nil, fmt.Errorf("key vault name, key name and key version are required")
	}
	kvClient := kv.New()
	err := kvClient.AddToUserAgent(version.GetUserAgent())
	if err != nil {
		return nil, fmt.Errorf("failed to add user agent to keyvault client, error: %+v", err)
	}
	env, err := auth.ParseAzureEnvironment(config.Cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cloud environment: %s, error: %+v", config.Cloud, err)
	}
	if proxyMode {
		env.ActiveDirectoryEndpoint = fmt.Sprintf("http://%s:%d/AzureActiveDirectory", proxyAddress, proxyPort)
	}

	token, err := auth.GetKeyvaultToken(config, env, proxyMode)
	if err != nil {
		return nil, fmt.Errorf("failed to get key vault token, error: %+v", err)
	}
	kvClient.Authorizer = token

	vaultURL, err := getVaultURL(vaultName, env)
	if err != nil {
		return nil, fmt.Errorf("failed to get vault url, error: %+v", err)
	}

	klog.InfoS("using kms key for encrypt/decrypt", "vaultURL", *vaultURL, "keyName", keyName, "keyVersion", keyVersion)

	if proxyMode {
		proxyEndpoint := fmt.Sprintf("http://%s:%d/KeyVault/%s", proxyAddress, proxyPort, (*vaultURL)[8:])
		klog.InfoS("proxy url", "url", proxyEndpoint)
		vaultURL = &proxyEndpoint
	}

	client := &keyVaultClient{
		baseClient:       kvClient,
		config:           config,
		vaultName:        vaultName,
		keyName:          keyName,
		keyVersion:       keyVersion,
		vaultURL:         *vaultURL,
		azureEnvironment: env,
	}
	return client, nil
}

func (kvc *keyVaultClient) Encrypt(ctx context.Context, cipher []byte) ([]byte, error) {
	value := base64.RawURLEncoding.EncodeToString(cipher)

	params := kv.KeyOperationsParameters{
		Algorithm: kv.RSA15,
		Value:     &value,
	}
	result, err := kvc.baseClient.Encrypt(ctx, kvc.vaultURL, kvc.keyName, kvc.keyVersion, params)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt, error: %+v", err)
	}
	return []byte(*result.Result), nil
}

func (kvc *keyVaultClient) Decrypt(ctx context.Context, plain []byte) ([]byte, error) {
	value := string(plain)

	params := kv.KeyOperationsParameters{
		Algorithm: kv.RSA15,
		Value:     &value,
	}

	result, err := kvc.baseClient.Decrypt(ctx, kvc.vaultURL, kvc.keyName, kvc.keyVersion, params)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt, error: %+v", err)
	}
	bytes, err := base64.RawURLEncoding.DecodeString(*result.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to base64 decode result, error: %+v", err)
	}
	return bytes, nil
}

func getVaultURL(vaultName string, azureEnvironment *azure.Environment) (vaultURL *string, err error) {
	// Key Vault name must be a 3-24 character string
	if len(vaultName) < 3 || len(vaultName) > 24 {
		return nil, fmt.Errorf("invalid vault name: %q, must be between 3 and 24 chars", vaultName)
	}

	// See docs for validation spec: https://docs.microsoft.com/en-us/azure/key-vault/about-keys-secrets-and-certificates#objects-identifiers-and-versioning
	isValid := regexp.MustCompile(`^[-A-Za-z0-9]+$`).MatchString
	if !isValid(vaultName) {
		return nil, fmt.Errorf("invalid vault name: %q, must match [-a-zA-Z0-9]{3,24}", vaultName)
	}

	vaultDNSSuffixValue := azureEnvironment.KeyVaultDNSSuffix
	vaultURI := "https://" + vaultName + "." + vaultDNSSuffixValue + "/"
	return &vaultURI, nil
}
