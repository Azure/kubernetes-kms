// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/Azure/kubernetes-kms/pkg/auth"
	"github.com/Azure/kubernetes-kms/pkg/config"
	"github.com/Azure/kubernetes-kms/pkg/consts"
	"github.com/Azure/kubernetes-kms/pkg/utils"
	"github.com/Azure/kubernetes-kms/pkg/version"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azkeys"
	"github.com/Azure/go-autorest/autorest/azure"
	"k8s.io/klog/v2"
)

// Client interface for interacting with Keyvault
type Client interface {
	Encrypt(ctx context.Context, cipher []byte) ([]byte, error)
	Decrypt(ctx context.Context, plain []byte) ([]byte, error)
}

type keyVaultClient struct {
	keysClient       *azkeys.Client
	config           *config.AzureConfig
	vaultName        string
	keyName          string
	keyVersion       string
	vaultURL         string
	azureEnvironment azure.Environment
}

// NewKeyVaultClient returns a new key vault client to use for kms operations
func newKeyVaultClient(
	config *config.AzureConfig,
	vaultName, keyName, keyVersion string,
	proxyMode bool,
	proxyAddress string,
	proxyPort int,
	managedHSM bool) (*keyVaultClient, error) {
	// Sanitize vaultName, keyName, keyVersion. (https://github.com/Azure/kubernetes-kms/issues/85)
	vaultName = utils.SanitizeString(vaultName)
	keyName = utils.SanitizeString(keyName)
	keyVersion = utils.SanitizeString(keyVersion)

	// this should be the case for bring your own key, clusters bootstrapped with
	// aks-engine or aks and standalone kms plugin deployments
	if len(vaultName) == 0 || len(keyName) == 0 || len(keyVersion) == 0 {
		return nil, fmt.Errorf("key vault name, key name and key version are required")
	}

	env, err := parseAzureEnvironment(config.Cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cloud environment: %s, error: %+v", config.Cloud, err)
	}
	if proxyMode {
		env.ActiveDirectoryEndpoint = fmt.Sprintf("http://%s:%d/", proxyAddress, proxyPort)
	}

	vaultResourceURL := getVaultResourceIdentifier(managedHSM, env)
	if vaultResourceURL == azure.NotAvailable {
		return nil, fmt.Errorf("keyvault resource identifier not available for cloud: %s", env.Name)
	}
	cred, err := auth.GetTokenCredential(config, env.ActiveDirectoryEndpoint, vaultResourceURL, proxyMode)
	if err != nil {
		return nil, fmt.Errorf("failed to get key vault token, error: %+v", err)
	}

	vaultURL, err := getVaultURL(vaultName, managedHSM, env)
	if err != nil {
		return nil, fmt.Errorf("failed to get vault url, error: %+v", err)
	}

	t := &transporter{}
	t.AddDecorator(SetUserAgent)

	if proxyMode {
		vaultURL = getProxiedVaultURL(vaultURL, proxyAddress, proxyPort)
		t.AddDecorator(SetProxyHeader)
	}

	klog.InfoS("using kms key for encrypt/decrypt", "vaultURL", *vaultURL, "keyName", keyName, "keyVersion", keyVersion)

	opts := &azkeys.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Transport: t,
		},
	}
	keysClient, err := azkeys.NewClient(*vaultURL, cred, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create keyvault client, error: %+v", err)
	}

	return &keyVaultClient{
		keysClient:       keysClient,
		config:           config,
		vaultName:        vaultName,
		keyName:          keyName,
		keyVersion:       keyVersion,
		vaultURL:         *vaultURL,
		azureEnvironment: env,
	}, nil
}

func (c *keyVaultClient) Encrypt(ctx context.Context, cipher []byte) ([]byte, error) {
	jsonWebKeyEncryptionAlgorithmRSA15 := azkeys.JSONWebKeyEncryptionAlgorithmRSA15
	params := azkeys.KeyOperationsParameters{
		Algorithm: &jsonWebKeyEncryptionAlgorithmRSA15,
		Value:     cipher,
	}

	response, err := c.keysClient.Encrypt(ctx, c.keyName, c.keyVersion, params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt, error: %+v", err)
	}
	return response.Result, nil
}

func (c *keyVaultClient) Decrypt(ctx context.Context, plain []byte) ([]byte, error) {
	jsonWebKeyEncryptionAlgorithmRSA15 := azkeys.JSONWebKeyEncryptionAlgorithmRSA15
	params := azkeys.KeyOperationsParameters{
		Algorithm: &jsonWebKeyEncryptionAlgorithmRSA15,
		Value:     plain,
	}

	response, err := c.keysClient.Decrypt(ctx, c.keyName, c.keyVersion, params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt, error: %+v", err)
	}
	return response.Result, nil
}

func getVaultURL(vaultName string, managedHSM bool, env azure.Environment) (vaultURL *string, err error) {
	// Key Vault name must be a 3-24 character string
	if len(vaultName) < 3 || len(vaultName) > 24 {
		return nil, fmt.Errorf("invalid vault name: %q, must be between 3 and 24 chars", vaultName)
	}

	// See docs for validation spec: https://docs.microsoft.com/en-us/azure/key-vault/about-keys-secrets-and-certificates#objects-identifiers-and-versioning
	isValid := regexp.MustCompile(`^[-A-Za-z0-9]+$`).MatchString
	if !isValid(vaultName) {
		return nil, fmt.Errorf("invalid vault name: %q, must match [-a-zA-Z0-9]{3,24}", vaultName)
	}

	vaultDNSSuffixValue := getVaultDNSSuffix(managedHSM, env)
	if vaultDNSSuffixValue == azure.NotAvailable {
		return nil, fmt.Errorf("vault dns suffix not available for cloud: %s", env.Name)
	}

	vaultURI := fmt.Sprintf("https://%s.%s/", vaultName, vaultDNSSuffixValue)
	return &vaultURI, nil
}

func getProxiedVaultURL(vaultURL *string, proxyAddress string, proxyPort int) *string {
	proxiedVaultURL := fmt.Sprintf("http://%s:%d/%s", proxyAddress, proxyPort, strings.TrimPrefix(*vaultURL, "https://"))
	return &proxiedVaultURL
}

func getVaultDNSSuffix(managedHSM bool, env azure.Environment) string {
	if managedHSM {
		return env.ManagedHSMDNSSuffix
	}
	return env.KeyVaultDNSSuffix
}

func getVaultResourceIdentifier(managedHSM bool, env azure.Environment) string {
	if managedHSM {
		return env.ResourceIdentifiers.ManagedHSM
	}
	return env.ResourceIdentifiers.KeyVault
}

// parseAzureEnvironment returns azure environment by name
func parseAzureEnvironment(cloudName string) (azure.Environment, error) {
	if cloudName == "" {
		return azure.PublicCloud, nil
	}
	return azure.EnvironmentFromName(cloudName)
}

type transporter struct {
	decorators []func(*http.Request)
}

func (t *transporter) AddDecorator(decorator func(*http.Request)) {
	t.decorators = append(t.decorators, decorator)
}

func SetUserAgent(req *http.Request) {
	req.Header.Set("User-Agent", version.GetUserAgent())
}

func SetProxyHeader(req *http.Request) {
	req.Header.Set(consts.RequestHeaderTargetType, consts.TargetTypeKeyVault)
}

func (t *transporter) Do(req *http.Request) (*http.Response, error) {
	for _, decorator := range t.decorators {
		decorator(req)
	}
	return http.DefaultTransport.RoundTrip(req)
}
