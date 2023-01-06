// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"fmt"
	"strings"
	"testing"

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
			config:           &config.AzureConfig{ClientID: "clientid", ClientSecret: "clientsecret", TenantID: "tenantid"},
			vaultName:        "testkv",
			keyName:          "key1",
			keyVersion:       "262067a9e8ba401aa8a746c5f1a7e147",
			proxyMode:        false,
			expectedVaultURL: "https://testkv.vault.azure.net/",
		},
		{
			desc:             "no error with double quotes",
			config:           &config.AzureConfig{ClientID: "clientid", ClientSecret: "clientsecret", TenantID: "tenantid"},
			vaultName:        "\"testkv\"",
			keyName:          "\"key1\"",
			keyVersion:       "\"262067a9e8ba401aa8a746c5f1a7e147\"",
			proxyMode:        false,
			expectedVaultURL: "https://testkv.vault.azure.net/",
		},
		// {
		// 	desc:             "no error with proxy mode",
		// 	config:           &config.AzureConfig{ClientID: "clientid", ClientSecret: "clientsecret", TenantID: "tenantid"},
		// 	vaultName:        "testkv",
		// 	keyName:          "key1",
		// 	keyVersion:       "262067a9e8ba401aa8a746c5f1a7e147",
		// 	proxyMode:        true,
		// 	proxyAddress:     "localhost",
		// 	proxyPort:        7788,
		// 	expectedVaultURL: "http://localhost:7788/testkv.vault.azure.net/",
		// },
		{
			desc:             "no error with managed hsm",
			config:           &config.AzureConfig{ClientID: "clientid", ClientSecret: "clientsecret", TenantID: "tenantid"},
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
				env, err := parseAzureEnvironment(testEnvs[idx])
				if err != nil {
					t.Fatalf("failed to parse azure environment from name, err: %+v", err)
				}
				if _, err = getVaultURL(test.vaultName, test.managedHSM, env); err == nil {
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
			env, err := parseAzureEnvironment(testEnvs[idx])
			if err != nil {
				t.Fatalf("failed to parse azure environment from name, err: %+v", err)
			}
			vaultURL, err := getVaultURL(vaultName, false, env)
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

func TestParseAzureEnvironment(t *testing.T) {
	envNamesArray := []string{"AZURECHINACLOUD", "AZUREGERMANCLOUD", "AZUREPUBLICCLOUD", "AZUREUSGOVERNMENTCLOUD", ""}
	for _, envName := range envNamesArray {
		azureEnv, err := parseAzureEnvironment(envName)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if strings.EqualFold(envName, "") && !strings.EqualFold(azureEnv.Name, "AZUREPUBLICCLOUD") {
			t.Fatalf("string doesn't match, expected AZUREPUBLICCLOUD, got %s", azureEnv.Name)
		} else if !strings.EqualFold(envName, "") && !strings.EqualFold(envName, azureEnv.Name) {
			t.Fatalf("string doesn't match, expected %s, got %s", envName, azureEnv.Name)
		}
	}

	wrongEnvName := "AZUREWRONGCLOUD"
	_, err := parseAzureEnvironment(wrongEnvName)
	if err == nil {
		t.Fatalf("expected error for wrong azure environment name")
	}
}
