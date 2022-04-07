// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/kubernetes-kms/pkg/auth"
	"github.com/Azure/kubernetes-kms/pkg/config"
)

var (
	testEnvs       = []string{"", "AZUREPUBLICCLOUD", "AZURECHINACLOUD", "AZUREGERMANCLOUD", "AZUREUSGOVERNMENTCLOUD"}
	vaultDNSSuffix = []string{"vault.azure.net", "vault.azure.net", "vault.azure.cn", "vault.microsoftazure.de", "vault.usgovcloudapi.net"}
)

func TestNewKeyVaultClientError(t *testing.T) {
	tests := []struct {
		desc         string
		config       *config.AzureConfig
		vaultName    string
		keyName      string
		keyVersion   string
		proxyMode    bool
		proxyAddress string
		proxyPort    int
		managedHSM   bool
	}{
		{
			desc:      "vault name not provided",
			config:    &config.AzureConfig{},
			proxyMode: false,
		},
		{
			desc:      "key name not provided",
			config:    &config.AzureConfig{},
			vaultName: "testkv",
			proxyMode: false,
		},
		{
			desc:      "key version not provided",
			config:    &config.AzureConfig{},
			vaultName: "testkv",
			keyName:   "k8s",
			proxyMode: false,
		},
		{
			desc:       "no credentials in config",
			config:     &config.AzureConfig{},
			vaultName:  "testkv",
			keyName:    "key1",
			keyVersion: "262067a9e8ba401aa8a746c5f1a7e147",
		},
		{
			desc:       "managed hsm not available in the azure environment",
			config:     &config.AzureConfig{ClientID: "clientid", ClientSecret: "clientsecret", Cloud: "AzureGermanCloud"},
			vaultName:  "testkv",
			keyName:    "key1",
			keyVersion: "262067a9e8ba401aa8a746c5f1a7e147",
			managedHSM: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			if _, err := newKeyVaultClient(test.config, test.vaultName, test.keyName, test.keyVersion, test.proxyMode, test.proxyAddress, test.proxyPort, test.managedHSM); err == nil {
				t.Fatalf("newKeyVaultClient() expected error, got nil")
			}
		})
	}
}

func TestNewKeyVaultClient(t *testing.T) {
	tests := []struct {
		desc             string
		config           *config.AzureConfig
		vaultName        string
		keyName          string
		keyVersion       string
		proxyMode        bool
		proxyAddress     string
		proxyPort        int
		managedHSM       bool
		expectedVaultURL string
	}{
		{
			desc:             "no error",
			config:           &config.AzureConfig{ClientID: "clientid", ClientSecret: "clientsecret"},
			vaultName:        "testkv",
			keyName:          "key1",
			keyVersion:       "262067a9e8ba401aa8a746c5f1a7e147",
			proxyMode:        false,
			expectedVaultURL: "https://testkv.vault.azure.net/",
		},
		{
			desc:             "no error with double quotes",
			config:           &config.AzureConfig{ClientID: "clientid", ClientSecret: "clientsecret"},
			vaultName:        "\"testkv\"",
			keyName:          "\"key1\"",
			keyVersion:       "\"262067a9e8ba401aa8a746c5f1a7e147\"",
			proxyMode:        false,
			expectedVaultURL: "https://testkv.vault.azure.net/",
		},
		{
			desc:             "no error with proxy mode",
			config:           &config.AzureConfig{ClientID: "clientid", ClientSecret: "clientsecret"},
			vaultName:        "testkv",
			keyName:          "key1",
			keyVersion:       "262067a9e8ba401aa8a746c5f1a7e147",
			proxyMode:        true,
			proxyAddress:     "localhost",
			proxyPort:        7788,
			expectedVaultURL: "http://localhost:7788/testkv.vault.azure.net/",
		},
		{
			desc:             "no error with managed hsm",
			config:           &config.AzureConfig{ClientID: "clientid", ClientSecret: "clientsecret"},
			vaultName:        "testkv",
			keyName:          "key1",
			keyVersion:       "262067a9e8ba401aa8a746c5f1a7e147",
			managedHSM:       true,
			proxyMode:        false,
			expectedVaultURL: "https://testkv.managedhsm.azure.net/",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			kvClient, err := newKeyVaultClient(test.config, test.vaultName, test.keyName, test.keyVersion, test.proxyMode, test.proxyAddress, test.proxyPort, test.managedHSM)
			if err != nil {
				t.Fatalf("newKeyVaultClient() failed with error: %v", err)
			}
			if kvClient == nil {
				t.Fatalf("newKeyVaultClient() expected kv client to not be nil")
			}
			if !strings.Contains(kvClient.baseClient.UserAgent, "k8s-kms-keyvault") {
				t.Fatalf("newKeyVaultClient() expected k8s-kms-keyvault user agent")
			}
			if kvClient.vaultURL != test.expectedVaultURL {
				t.Fatalf("expected vault URL: %v, got vault URL: %v", test.expectedVaultURL, kvClient.vaultURL)
			}
		})
	}
}

func TestGetVaultURLError(t *testing.T) {
	tests := []struct {
		desc       string
		vaultName  string
		managedHSM bool
	}{
		{
			desc:      "vault name > 24",
			vaultName: "longkeyvaultnamewhichisnotvalid",
		},
		{
			desc:      "vault name < 3",
			vaultName: "kv",
		},
		{
			desc:      "vault name contains non alpha-numeric chars",
			vaultName: "kv_test",
		},
	}

	for _, test := range tests {
		for idx := range testEnvs {
			t.Run(fmt.Sprintf("%s/%s", test.desc, testEnvs[idx]), func(t *testing.T) {
				azEnv, err := auth.ParseAzureEnvironment(testEnvs[idx])
				if err != nil {
					t.Fatalf("failed to parse azure environment from name, err: %+v", err)
				}
				if _, err = getVaultURL(test.vaultName, test.managedHSM, azEnv); err == nil {
					t.Fatalf("getVaultURL() expected error, got nil")
				}
			})
		}
	}
}

func TestGetVaultURL(t *testing.T) {
	vaultName := "testkv"

	for idx := range testEnvs {
		t.Run(testEnvs[idx], func(t *testing.T) {
			azEnv, err := auth.ParseAzureEnvironment(testEnvs[idx])
			if err != nil {
				t.Fatalf("failed to parse azure environment from name, err: %+v", err)
			}
			vaultURL, err := getVaultURL(vaultName, false, azEnv)
			if err != nil {
				t.Fatalf("expected no error of getting vault URL, got error: %v", err)
			}
			expectedURL := "https://" + vaultName + "." + vaultDNSSuffix[idx] + "/"
			if expectedURL != *vaultURL {
				t.Fatalf("expected vault url: %s, got: %s", expectedURL, *vaultURL)
			}
		})
	}
}
