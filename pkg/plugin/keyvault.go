// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/kubernetes-kms/pkg/auth"
	"github.com/Azure/kubernetes-kms/pkg/config"
	"github.com/Azure/kubernetes-kms/pkg/consts"
	"github.com/Azure/kubernetes-kms/pkg/utils"
	"github.com/Azure/kubernetes-kms/pkg/version"

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys"
	"k8s.io/kms/pkg/service"
	"monis.app/mlog"
)

// encryptionResponseVersion is validated prior to decryption.
// This is helpful in case we want to change anything about the data we send in the future.
var encryptionResponseVersion = "1"

const (
	dateAnnotationKey             = "date.azure.akv.io"
	requestIDAnnotationKey        = "x-ms-request-id.azure.akv.io"
	keyvaultRegionAnnotationKey   = "x-ms-keyvault-region.azure.akv.io"
	versionAnnotationKey          = "version.azure.akv.io"
	algorithmAnnotationKey        = "algorithm.azure.akv.io"
	dateAnnotationValue           = "Date"
	requestIDAnnotationValue      = "X-Ms-Request-Id"
	keyvaultRegionAnnotationValue = "X-Ms-Keyvault-Region"
)

// Client interface for interacting with Keyvault.
type Client interface {
	Encrypt(
		ctx context.Context,
		plain []byte,
		encryptionAlgorithm azkeys.EncryptionAlgorithm,
	) (*service.EncryptResponse, error)
	Decrypt(
		ctx context.Context,
		cipher []byte,
		encryptionAlgorithm azkeys.EncryptionAlgorithm,
		apiVersion string,
		annotations map[string][]byte,
		decryptRequestKeyID string,
	) ([]byte, error)
}

// KeyVaultClient is a client for interacting with Keyvault.
type KeyVaultClient struct {
	baseClient *azkeys.Client
	config     *config.AzureConfig
	vaultName  string
	keyName    string
	keyVersion string
	keyIDHash  string
}

// NewKeyVaultClient returns a new key vault client to use for kms operations.
func NewKeyVaultClient(
	config *config.AzureConfig,
	vaultName, keyName, keyVersion string,
	proxyMode bool,
	proxyAddress string,
	proxyPort int,
	managedHSM bool,
) (Client, error) {
	// Sanitize vaultName, keyName, keyVersion. (https://github.com/Azure/kubernetes-kms/issues/85)
	vaultName = utils.SanitizeString(vaultName)
	keyName = utils.SanitizeString(keyName)
	keyVersion = utils.SanitizeString(keyVersion)

	// this should be the case for bring your own key, clusters bootstrapped with
	// aks-engine or aks and standalone kms plugin deployments
	if len(vaultName) == 0 || len(keyName) == 0 || len(keyVersion) == 0 {
		return nil, fmt.Errorf("key vault name, key name and key version are required")
	}

	vaultURL, err := getVaultURL(vaultName, managedHSM, config.Cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to get vault url, error: %+v", err)
	}
	if proxyMode {
		vaultURL = getProxiedVaultURL(vaultURL, proxyAddress, proxyPort)
	}

	aadEndpoint, err := getAadEndpoint(config, proxyMode, proxyAddress, proxyPort)
	if err != nil {
		return nil, fmt.Errorf("failed to get aad endpoint: %v", err)
	}

	token, err := auth.GetKeyvaultToken(config, aadEndpoint, proxyMode)
	if err != nil {
		return nil, fmt.Errorf("failed to get keyvault token: %v", err)
	}

	kvClient, err := azkeys.NewClient(vaultURL, token, &azkeys.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			PerCallPolicies: []policy.Policy{policyFunc(func(req *policy.Request) (*http.Response, error) {
				req.Raw().Header.Set("User-Agent", version.GetUserAgent())
				req.Raw().Header.Set(consts.RequestHeaderTargetType, consts.TargetTypeKeyVault)
				return req.Next()
			})},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create keyvault client: %v", err)
	}

	keyIDHash, err := getKeyIDHash(vaultURL, keyName, keyVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get key id hash, error: %w", err)
	}

	mlog.Always("using kms key for encrypt/decrypt", "vaultURL", vaultURL, "keyName", keyName, "keyVersion", keyVersion)

	client := &KeyVaultClient{
		baseClient: kvClient,
		config:     config,
		vaultName:  vaultName,
		keyName:    keyName,
		keyVersion: keyVersion,
		keyIDHash:  keyIDHash,
	}
	return client, nil
}

type policyFunc func(req *policy.Request) (*http.Response, error)

func (p policyFunc) Do(req *policy.Request) (*http.Response, error) {
	return p(req)
}

var _ policy.Policy = (*policyFunc)(nil)

// Encrypt encrypts the given plain text using the keyvault key.
func (kvc *KeyVaultClient) Encrypt(
	ctx context.Context,
	plain []byte,
	encryptionAlgorithm azkeys.EncryptionAlgorithm,
) (*service.EncryptResponse, error) {
	value := base64.RawURLEncoding.EncodeToString(plain)

	params := azkeys.KeyOperationParameters{
		Algorithm: &encryptionAlgorithm,
		Value:     []byte(value),
	}
	result, err := kvc.baseClient.Encrypt(ctx, kvc.keyName, kvc.keyVersion, params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt, error: %+v", err)
	}

	if kvc.keyIDHash != fmt.Sprintf("%x", sha256.Sum256([]byte(*result.KID))) {
		return nil, fmt.Errorf(
			"key id initialized does not match with the key id from encryption result, expected: %s, got: %s",
			kvc.keyIDHash,
			*result.KID,
		)
	}

	annotations := map[string][]byte{
		// dateAnnotationKey:           []byte(result.Header.Get(dateAnnotationValue)),
		// requestIDAnnotationKey:      []byte(result.Header.Get(requestIDAnnotationValue)),
		// keyvaultRegionAnnotationKey: []byte(result.Header.Get(keyvaultRegionAnnotationValue)),
		versionAnnotationKey:   []byte(encryptionResponseVersion),
		algorithmAnnotationKey: []byte(encryptionAlgorithm),
	}

	return &service.EncryptResponse{
		Ciphertext:  result.Result,
		KeyID:       kvc.keyIDHash,
		Annotations: annotations,
	}, nil
}

// Decrypt decrypts the given cipher text using the keyvault key.
func (kvc *KeyVaultClient) Decrypt(
	ctx context.Context,
	cipher []byte,
	encryptionAlgorithm azkeys.EncryptionAlgorithm,
	apiVersion string,
	annotations map[string][]byte,
	decryptRequestKeyID string,
) ([]byte, error) {
	if apiVersion == version.KMSv2APIVersion {
		err := kvc.validateAnnotations(annotations, decryptRequestKeyID, encryptionAlgorithm)
		if err != nil {
			return nil, err
		}
	}

	value := string(cipher)
	params := azkeys.KeyOperationParameters{
		Algorithm: &encryptionAlgorithm,
		Value:     []byte(value),
	}

	result, err := kvc.baseClient.Decrypt(ctx, kvc.keyName, kvc.keyVersion, params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt, error: %+v", err)
	}
	bytes, err := base64.RawURLEncoding.DecodeString(string(result.Result))
	if err != nil {
		return nil, fmt.Errorf("failed to base64 decode result, error: %+v", err)
	}

	return bytes, nil
}

// ValidateAnnotations validates following annotations before decryption:
// - Algorithm.
// - Version.
// It also validates keyID that the API server checks.
func (kvc *KeyVaultClient) validateAnnotations(
	annotations map[string][]byte,
	keyID string,
	encryptionAlgorithm azkeys.EncryptionAlgorithm,
) error {
	if len(annotations) == 0 {
		return fmt.Errorf("invalid annotations, annotations cannot be empty")
	}

	if keyID != kvc.keyIDHash {
		return fmt.Errorf(
			"key id %s does not match expected key id %s used for encryption",
			keyID,
			kvc.keyIDHash,
		)
	}

	algorithm := string(annotations[algorithmAnnotationKey])
	if algorithm != string(encryptionAlgorithm) {
		return fmt.Errorf(
			"algorithm %s does not match expected algorithm %s used for encryption",
			algorithm,
			encryptionAlgorithm,
		)
	}

	version := string(annotations[versionAnnotationKey])
	if version != encryptionResponseVersion {
		return fmt.Errorf(
			"version %s does not match expected version %s used for encryption",
			version,
			encryptionResponseVersion,
		)
	}

	return nil
}

func getVaultURL(vaultName string, managedHSM bool, cloud string) (vaultURL string, err error) {
	// Key Vault name must be a 3-24 character string
	if len(vaultName) < 3 || len(vaultName) > 24 {
		return "", fmt.Errorf("invalid vault name: %q, must be between 3 and 24 chars", vaultName)
	}

	// See docs for validation spec: https://docs.microsoft.com/en-us/azure/key-vault/about-keys-secrets-and-certificates#objects-identifiers-and-versioning
	isValid := regexp.MustCompile(`^[-A-Za-z0-9]+$`).MatchString
	if !isValid(vaultName) {
		return "", fmt.Errorf("invalid vault name: %q, must match [-a-zA-Z0-9]{3,24}", vaultName)
	}

	vaultDNSSuffixValue, err := getVaultDNSSuffix(managedHSM, cloud)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("https://%s.%s/", vaultName, vaultDNSSuffixValue), nil
}

func getProxiedVaultURL(vaultURL string, proxyAddress string, proxyPort int) string {
	return fmt.Sprintf("http://%s:%d/%s", proxyAddress, proxyPort, strings.TrimPrefix(vaultURL, "https://"))
}

func getVaultDNSSuffix(managedHSM bool, cloud string) (string, error) {
	if managedHSM {
		switch {
		case strings.EqualFold(cloud, "AzurePublicCloud"), strings.EqualFold(cloud, "AzureCloud"), cloud == "":
			return "https://managedhsm.azure.net/", nil
		case strings.EqualFold(cloud, "AzureChinaCloud"):
			return "", fmt.Errorf("no HSM endpoint in cloud %s", cloud)
		case strings.EqualFold(cloud, "AzureGovernmentCloud"):
			return "", fmt.Errorf("no HSM endpoint in cloud %s", cloud)
		default:
			return "", fmt.Errorf("unknown cloud %s", cloud)
		}
	}
	switch {
	case strings.EqualFold(cloud, "AzurePublicCloud"), strings.EqualFold(cloud, "AzureCloud"), cloud == "":
		return ".vault.azure.net", nil
	case strings.EqualFold(cloud, "AzureChinaCloud"):
		return ".vault.azure.cn", nil
	case strings.EqualFold(cloud, "AzureGovernmentCloud"):
		return ".vault.usgovcloudapi.net", nil
	default:
		return "", fmt.Errorf("unknown cloud %s", cloud)
	}
}

func getKeyIDHash(vaultURL, keyName, keyVersion string) (string, error) {
	if vaultURL == "" || keyName == "" || keyVersion == "" {
		return "", fmt.Errorf("vault url, key name and key version cannot be empty")
	}

	baseURL, err := url.Parse(vaultURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse vault url, error: %w", err)
	}

	urlPath := path.Join("keys", keyName, keyVersion)
	keyID := baseURL.ResolveReference(
		&url.URL{
			Path: urlPath,
		},
	).String()

	return fmt.Sprintf("%x", sha256.Sum256([]byte(keyID))), nil
}

func getAadEndpoint(azureConfig *config.AzureConfig, proxyMode bool, proxyAddress string, proxyPort int) (string, error) {
	if proxyMode {
		return fmt.Sprintf("http://%s:%d/", proxyAddress, proxyPort), nil
	}
	switch {
	case strings.EqualFold(azureConfig.Cloud, "AzurePublicCloud"), strings.EqualFold(azureConfig.Cloud, "AzureCloud"), azureConfig.Cloud == "":
		return cloud.AzurePublic.ActiveDirectoryAuthorityHost, nil
	case strings.EqualFold(azureConfig.Cloud, "AzureChinaCloud"):
		return cloud.AzureChina.ActiveDirectoryAuthorityHost, nil
	case strings.EqualFold(azureConfig.Cloud, "AzureGovernmentCloud"):
		return cloud.AzureGovernment.ActiveDirectoryAuthorityHost, nil
	}
	return "", fmt.Errorf("invalid cloud type %s", azureConfig.Cloud)
}
