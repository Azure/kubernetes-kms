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
	"strings"

	"github.com/Azure/kubernetes-kms/pkg/auth"
	"github.com/Azure/kubernetes-kms/pkg/config"

	kv "github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	kvmgmt "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	storagemgmt "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/klog/v2"
)

// Client interface for interacting with Keyvault
type Client interface {
	Encrypt(ctx context.Context, cipher []byte) ([]byte, error)
	Decrypt(ctx context.Context, plain []byte) ([]byte, error)
	CheckIfKeyExists(ctx context.Context) error
}

type keyVaultClient struct {
	baseClient       kv.BaseClient
	config           *config.AzureConfig
	vaultName        string
	vaultSKU         string
	keyName          string
	keyVersion       string
	vaultURL         string
	azureEnvironment *azure.Environment
}

// NewKeyVaultClient returns a new key vault client to use for kms operations
func newKeyVaultClient(config *config.AzureConfig, vaultName, keyName, keyVersion, vaultSKU string) (*keyVaultClient, error) {
	// this should be the case for bring your own key, clusters bootstrapped with
	// aks-engine or aks and standalone kms plugin deployments
	if len(vaultName) == 0 || len(keyName) == 0 {
		return nil, fmt.Errorf("key vault name and key name are required")
	}
	kvClient := kv.New()
	err := kvClient.AddToUserAgent("k8s-kms-keyvault")
	if err != nil {
		return nil, fmt.Errorf("failed to add user agent to keyvault client, error: %+v", err)
	}
	env, err := auth.ParseAzureEnvironment(config.Cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cloud environment: %s, error: %+v", config.Cloud, err)
	}
	token, err := auth.GetKeyvaultToken(config, env)
	if err != nil {
		return nil, fmt.Errorf("failed to get key vault token, error: %+v", err)
	}
	kvClient.Authorizer = token

	vaultURL, err := getVaultURL(vaultName, env)
	if err != nil {
		return nil, fmt.Errorf("failed to get vault url, error: %+v", err)
	}

	client := &keyVaultClient{
		baseClient:       kvClient,
		config:           config,
		vaultName:        vaultName,
		vaultSKU:         vaultSKU,
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

func (kvc *keyVaultClient) CheckIfKeyExists(ctx context.Context) error {
	keyBundle, err := kvc.baseClient.GetKey(ctx, kvc.vaultURL, kvc.keyName, kvc.keyVersion)
	// key version provided, then check key with that exact version exists
	// if not exists, then fail as this is configured as part of deployment
	if len(kvc.keyVersion) != 0 {
		if err != nil {
			return err
		}
		if getKeyVersionFromKID(*keyBundle.Key.Kid) != kvc.keyVersion {
			return fmt.Errorf("key with version: %s not found", kvc.keyVersion)
		}
	}
	// key version not provided but the key exists in the key vault
	if err == nil {
		// populate the latest key version from the result we got
		kvc.keyVersion = getKeyVersionFromKID(*keyBundle.Key.Kid)
		klog.InfoS("using kms key for encrypt/decrypt", "vaultName", kvc.vaultName, "keyName", kvc.keyName, "keyVersion", kvc.keyVersion)
		return nil
	}
	// key with given name doesn't exist
	// no version provided, so create a new key with the name
	kvc.keyVersion, err = kvc.createKey(ctx)
	if err != nil {
		return fmt.Errorf("failed to create new key, error: %+v", err)
	}
	return nil
}

func (kvc *keyVaultClient) createKey(ctx context.Context) (string, error) {
	klog.InfoS("creating new key", "key", kvc.keyName, "vaultName", kvc.vaultName)
	storageAccountsClient := storagemgmt.NewAccountsClientWithBaseURI(kvc.azureEnvironment.ResourceManagerEndpoint, kvc.config.SubscriptionID)
	token, err := auth.GetManagementToken(kvc.config, kvc.azureEnvironment)
	if err != nil {
		return "", fmt.Errorf("failed to create storage client, error: %+v", err)
	}
	storageAccountsClient.Authorizer = token
	storageAcctName := kvc.vaultName
	res, err := storageAccountsClient.ListKeys(ctx, kvc.config.ResourceGroupName, storageAcctName, storagemgmt.Kerb)
	if err != nil {
		return "", err
	}
	storageKey := *(((*res.Keys)[0]).Value)
	var storageCli storage.Client
	if kvc.azureEnvironment.Name == azure.PublicCloud.Name {
		storageCli, err = storage.NewBasicClient(storageAcctName, storageKey)
	} else {
		storageCli, err = storage.NewBasicClientOnSovereignCloud(storageAcctName, storageKey, *kvc.azureEnvironment)
	}
	if err != nil {
		return "", err
	}
	blobCli := storageCli.GetBlobService()
	// Get container
	cnt := blobCli.GetContainerReference(kvc.keyName)
	ok, err := cnt.Exists()
	if err != nil {
		return "", err
	}
	if !ok {
		klog.InfoS("creating container", "name", kvc.keyName)
		// Create container
		options := storage.CreateContainerOptions{
			Access: storage.ContainerAccessTypeContainer,
		}
		_, err := cnt.CreateIfNotExists(&options)
		if err != nil {
			return "", err
		}
	}
	// Get blob
	b := cnt.GetBlobReference(kvc.keyName)
	ok, err = b.Exists()
	if !ok {
		klog.InfoS("creating blob", "name", kvc.keyName)
		// Create blob
		err = b.CreateBlockBlob(nil)
		if err != nil {
			return "", err
		}
	}
	// Acquiring lease on blob, if blob already has a lease, return err
	_, err = b.AcquireLease(60, "", nil)
	if err != nil {
		return "", err
	}
	keyType := kv.RSA
	if strings.EqualFold(kvc.vaultSKU, string(kvmgmt.Premium)) {
		keyType = kv.RSAHSM
	}
	key, err := kvc.baseClient.CreateKey(
		ctx,
		kvc.vaultURL,
		kvc.keyName,
		kv.KeyCreateParameters{
			KeyAttributes: &kv.KeyAttributes{
				Enabled: to.BoolPtr(true),
			},
			KeySize: to.Int32Ptr(2048),
			KeyOps: &[]kv.JSONWebKeyOperation{
				kv.Encrypt,
				kv.Decrypt,
			},
			Kty: keyType,
		})
	if err != nil {
		return "", err
	}
	keyVersion := getKeyVersionFromKID(*key.Key.Kid)
	klog.InfoS("created kms key", "name", kvc.keyName, "version", keyVersion)
	return keyVersion, nil
}

func getVaultURL(vaultName string, azureEnvironment *azure.Environment) (vaultURL *string, err error) {
	klog.V(2).Infof("vaultName: %s", vaultName)

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

// getKeyVersionFromKID parses the id to retrieve the version
// of key fetched
// example id format - https://kindkv.vault.azure.net/secrets/actual/1f304204f3624873aab40231241243eb
// TODO (aramase) follow up on https://github.com/Azure/azure-rest-api-specs/issues/10825 to provide
// a native way to obtain the version
func getKeyVersionFromKID(id string) string {
	splitID := strings.Split(id, "/")
	return splitID[len(splitID)-1]
}
