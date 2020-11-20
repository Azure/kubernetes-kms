// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"strings"
	"testing"

	"github.com/Azure/kubernetes-kms/pkg/auth"
	"github.com/Azure/kubernetes-kms/pkg/config"
)

func TestNewKeyVaultClient(t *testing.T) {
	tests := []struct {
		desc        string
		config      *config.AzureConfig
		vaultName   string
		keyName     string
		keyVersion  string
		vaultSKU    string
		expectedErr bool
	}{
		{
			desc:        "vault name not provided",
			config:      &config.AzureConfig{},
			expectedErr: true,
		},
		{
			desc:        "key name not provided",
			config:      &config.AzureConfig{},
			vaultName:   "testkv",
			expectedErr: true,
		},
		{
			desc:        "no credentials in config",
			config:      &config.AzureConfig{},
			vaultName:   "testkv",
			keyName:     "key1",
			expectedErr: true,
		},
		{
			desc:        "no error",
			config:      &config.AzureConfig{ClientID: "clientid", ClientSecret: "clientsecret"},
			vaultName:   "testkv",
			keyName:     "key1",
			expectedErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			kvClient, err := newKeyVaultClient(test.config, test.vaultName, test.keyName, test.keyVersion, test.vaultSKU)
			if test.expectedErr && err == nil || !test.expectedErr && err != nil {
				t.Fatalf("expected error: %v, got error: %v", test.expectedErr, err)
			}
			if !test.expectedErr {
				if kvClient == nil {
					t.Fatalf("expected kv client to not be nil")
				}
				if !strings.Contains(kvClient.baseClient.UserAgent, "k8s-kms-keyvault") {
					t.Fatalf("expected k8s-kms-keyvault user agent")
				}
			}
		})
	}
}

func TestGetVaultURL(t *testing.T) {
	testEnvs := []string{"", "AZUREPUBLICCLOUD", "AZURECHINACLOUD", "AZUREGERMANCLOUD", "AZUREUSGOVERNMENTCLOUD"}
	vaultDNSSuffix := []string{"vault.azure.net", "vault.azure.net", "vault.azure.cn", "vault.microsoftazure.de", "vault.usgovcloudapi.net"}

	tests := []struct {
		desc        string
		vaultName   string
		expectedErr bool
	}{
		{
			desc:        "vault name > 24",
			vaultName:   "longkeyvaultnamewhichisnotvalid",
			expectedErr: true,
		},
		{
			desc:        "vault name < 3",
			vaultName:   "kv",
			expectedErr: true,
		},
		{
			desc:        "vault name contains non alpha-numeric chars",
			vaultName:   "kv_test",
			expectedErr: true,
		},
		{
			desc:        "valid vault name in public cloud",
			vaultName:   "testkv",
			expectedErr: false,
		},
	}

	for idx, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			azEnv, err := auth.ParseAzureEnvironment(testEnvs[idx])
			if err != nil {
				t.Fatalf("failed to parse azure environment from name, err: %+v", err)
			}
			vaultURL, err := getVaultURL(test.vaultName, azEnv)
			if test.expectedErr && err == nil || !test.expectedErr && err != nil {
				t.Fatalf("expected error: %v, got error: %v", test.expectedErr, err)
			}
			expectedURL := "https://" + test.vaultName + "." + vaultDNSSuffix[idx] + "/"
			if !test.expectedErr && expectedURL != *vaultURL {
				t.Fatalf("expected vault url: %s, got: %s", expectedURL, *vaultURL)
			}
		})
	}
}
