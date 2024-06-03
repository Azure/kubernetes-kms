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
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/Azure/kubernetes-kms/pkg/auth"
	"github.com/Azure/kubernetes-kms/pkg/config"
	"github.com/Azure/kubernetes-kms/pkg/consts"
	"github.com/Azure/kubernetes-kms/pkg/utils"
	"github.com/Azure/kubernetes-kms/pkg/version"

	kv "github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
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
		encryptionAlgorithm kv.JSONWebKeyEncryptionAlgorithm,
	) (*service.EncryptResponse, error)
	Decrypt(
		ctx context.Context,
		cipher []byte,
		encryptionAlgorithm kv.JSONWebKeyEncryptionAlgorithm,
		apiVersion string,
		annotations map[string][]byte,
		decryptRequestKeyID string,
	) ([]byte, error)
	GetUserAgent() string
	GetVaultURL() string
}

// KeyVaultClient is a client for interacting with Keyvault.
type KeyVaultClient struct {
	baseClient       kv.BaseClient
	config           *config.AzureConfig
	vaultName        string
	keyName          string
	keyVersion       string
	vaultURL         string
	keyIDHash        string
	azureEnvironment *azure.Environment
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
		env.ActiveDirectoryEndpoint = fmt.Sprintf("http://%s:%d/", proxyAddress, proxyPort)
	}

	vaultResourceURL := getVaultResourceIdentifier(managedHSM, env)
	if vaultResourceURL == azure.NotAvailable {
		return nil, fmt.Errorf("keyvault resource identifier not available for cloud: %s", env.Name)
	}
	token, err := auth.GetKeyvaultToken(config, env, vaultResourceURL, proxyMode)
	if err != nil {
		return nil, fmt.Errorf("failed to get key vault token, error: %+v", err)
	}
	kvClient.Authorizer = token

	vaultURL, err := getVaultURL(vaultName, managedHSM, env)
	if err != nil {
		return nil, fmt.Errorf("failed to get vault url, error: %+v", err)
	}

	keyIDHash, err := getKeyIDHash(*vaultURL, keyName, keyVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get key id hash, error: %w", err)
	}

	if proxyMode {
		kvClient.RequestInspector = autorest.WithHeader(consts.RequestHeaderTargetType, consts.TargetTypeKeyVault)
		vaultURL = getProxiedVaultURL(vaultURL, proxyAddress, proxyPort)
	}

	mlog.Always("using kms key for encrypt/decrypt", "vaultURL", *vaultURL, "keyName", keyName, "keyVersion", keyVersion)

	client := &KeyVaultClient{
		baseClient:       kvClient,
		config:           config,
		vaultName:        vaultName,
		keyName:          keyName,
		keyVersion:       "",
		vaultURL:         *vaultURL,
		azureEnvironment: env,
		keyIDHash:        keyIDHash,
	}
	return client, nil
}

// Encrypt encrypts the given plain text using the keyvault key.
func (kvc *KeyVaultClient) Encrypt(
	ctx context.Context,
	plain []byte,
	encryptionAlgorithm kv.JSONWebKeyEncryptionAlgorithm,
) (*service.EncryptResponse, error) {
	value := base64.RawURLEncoding.EncodeToString(plain)

	params := kv.KeyOperationsParameters{
		Algorithm: encryptionAlgorithm,
		Value:     &value,
	}
	result, err := kvc.baseClient.Encrypt(ctx, kvc.vaultURL, kvc.keyName, kvc.keyVersion, params)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt, error: %+v", err)
	}

	if kvc.keyIDHash != fmt.Sprintf("%x", sha256.Sum256([]byte(*result.Kid))) {
		return nil, fmt.Errorf(
			"key id initialized does not match with the key id from encryption result, expected: %s, got: %s",
			kvc.keyIDHash,
			*result.Kid,
		)
	}

	annotations := map[string][]byte{
		dateAnnotationKey:           []byte(result.Header.Get(dateAnnotationValue)),
		requestIDAnnotationKey:      []byte(result.Header.Get(requestIDAnnotationValue)),
		keyvaultRegionAnnotationKey: []byte(result.Header.Get(keyvaultRegionAnnotationValue)),
		versionAnnotationKey:        []byte(encryptionResponseVersion),
		algorithmAnnotationKey:      []byte(encryptionAlgorithm),
	}

	return &service.EncryptResponse{
		Ciphertext:  []byte(*result.Result),
		KeyID:       kvc.keyIDHash,
		Annotations: annotations,
	}, nil
}

// Decrypt decrypts the given cipher text using the keyvault key.
func (kvc *KeyVaultClient) Decrypt(
	ctx context.Context,
	cipher []byte,
	encryptionAlgorithm kv.JSONWebKeyEncryptionAlgorithm,
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
	params := kv.KeyOperationsParameters{
		Algorithm: encryptionAlgorithm,
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

func (kvc *KeyVaultClient) GetUserAgent() string {
	return kvc.baseClient.UserAgent
}

func (kvc *KeyVaultClient) GetVaultURL() string {
	return kvc.vaultURL
}

// ValidateAnnotations validates following annotations before decryption:
// - Algorithm.
// - Version.
// It also validates keyID that the API server checks.
func (kvc *KeyVaultClient) validateAnnotations(
	annotations map[string][]byte,
	keyID string,
	encryptionAlgorithm kv.JSONWebKeyEncryptionAlgorithm,
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

func getVaultURL(vaultName string, managedHSM bool, env *azure.Environment) (vaultURL *string, err error) {
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

func getVaultDNSSuffix(managedHSM bool, env *azure.Environment) string {
	if managedHSM {
		return env.ManagedHSMDNSSuffix
	}
	return env.KeyVaultDNSSuffix
}

func getVaultResourceIdentifier(managedHSM bool, env *azure.Environment) string {
	if managedHSM {
		return env.ResourceIdentifiers.ManagedHSM
	}
	return env.ResourceIdentifiers.KeyVault
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
