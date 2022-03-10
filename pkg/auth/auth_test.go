// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package auth

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/kubernetes-kms/pkg/config"
)

func TestParseAzureEnvironment(t *testing.T) {
	envNamesArray := []string{"AZURECHINACLOUD", "AZUREGERMANCLOUD", "AZUREPUBLICCLOUD", "AZUREUSGOVERNMENTCLOUD", ""}
	for _, envName := range envNamesArray {
		azureEnv, err := ParseAzureEnvironment(envName)
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
	_, err := ParseAzureEnvironment(wrongEnvName)
	if err == nil {
		t.Fatalf("expected error for wrong azure environment name")
	}
}

func TestRedactClientCredentials(t *testing.T) {
	tests := []struct {
		name     string
		clientID string
		expected string
	}{
		{
			name:     "should redact client id",
			clientID: "aabc0000-a83v-9h4m-000j-2c0a66b0c1f9",
			expected: "aabc##### REDACTED #####c1f9",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := redactClientCredentials(test.clientID)
			if actual != test.expected {
				t.Fatalf("expected: %s, got %s", test.expected, actual)
			}
		})
	}
}

func TestGetServicePrincipalTokenFromMSIWithUserAssignedID(t *testing.T) {
	tests := []struct {
		name      string
		config    *config.AzureConfig
		proxyMode bool // The proxy mode doesn't matter if user-assigned managed identity is used to get service principal token
	}{
		{
			name: "using user-assigned managed identity to access keyvault",
			config: &config.AzureConfig{
				UseManagedIdentityExtension: true,
				UserAssignedIdentityID:      "clientID",
				TenantID:                    "TenantID",
				ClientID:                    "AADClientID",
				ClientSecret:                "AADClientSecret",
			},
			proxyMode: false,
		},
		// The Azure service principal is ignored when
		// UseManagedIdentityExtension is set to true
		{
			name: "using user-assigned managed identity over service principal if set to true",
			config: &config.AzureConfig{
				UseManagedIdentityExtension: true,
				UserAssignedIdentityID:      "clientID",
			},
			proxyMode: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token, err := GetServicePrincipalToken(test.config, "https://login.microsoftonline.com/", "https://vault.azure.net", test.proxyMode)
			if err != nil {
				t.Fatalf("expected err to be nil, got: %v", err)
			}
			msiEndpoint, err := adal.GetMSIVMEndpoint()
			if err != nil {
				t.Fatalf("expected err to be nil, got: %v", err)
			}
			spt, err := adal.NewServicePrincipalTokenFromMSIWithUserAssignedID(msiEndpoint, "https://vault.azure.net", "clientID")
			if err != nil {
				t.Fatalf("expected err to be nil, got: %v", err)
			}
			if !reflect.DeepEqual(token, spt) {
				t.Fatalf("expected: %v, got: %v", spt, token)
			}
		})
	}
}

func TestGetServicePrincipalTokenFromMSI(t *testing.T) {
	tests := []struct {
		name      string
		config    *config.AzureConfig
		proxyMode bool // The proxy mode doesn't matter if MSI is used to get service principal token
	}{
		{
			name: "using system-assigned managed identity to access keyvault",
			config: &config.AzureConfig{
				UseManagedIdentityExtension: true,
			},
			proxyMode: false,
		},
		// The Azure service principal is ignored when
		// UseManagedIdentityExtension is set to true
		{
			name: "using system-assigned managed identity over service principal if set to true",
			config: &config.AzureConfig{
				UseManagedIdentityExtension: true,
				TenantID:                    "TenantID",
				ClientID:                    "AADClientID",
				ClientSecret:                "AADClientSecret",
			},
			proxyMode: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token, err := GetServicePrincipalToken(test.config, "https://login.microsoftonline.com/", "https://vault.azure.net", test.proxyMode)
			if err != nil {
				t.Fatalf("expected err to be nil, got: %v", err)
			}
			msiEndpoint, err := adal.GetMSIVMEndpoint()
			if err != nil {
				t.Fatalf("expected err to be nil, got: %v", err)
			}
			spt, err := adal.NewServicePrincipalTokenFromMSI(msiEndpoint, "https://vault.azure.net")
			if err != nil {
				t.Fatalf("expected err to be nil, got: %v", err)
			}
			if !reflect.DeepEqual(token, spt) {
				t.Fatalf("expected: %v, got: %v", spt, token)
			}
		})
	}
}

func TestGetServicePrincipalToken(t *testing.T) {
	tests := []struct {
		name   string
		config *config.AzureConfig
	}{
		{
			name: "using service-principal credentials to access keyvault",
			config: &config.AzureConfig{
				TenantID:     "TenantID",
				ClientID:     "AADClientID",
				ClientSecret: "AADClientSecret",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token, err := GetServicePrincipalToken(test.config, "https://login.microsoftonline.com/", "https://vault.azure.net", false)
			if err != nil {
				t.Fatalf("expected err to be nil, got: %v", err)
			}
			env := &azure.PublicCloud

			oauthConfig, err := adal.NewOAuthConfig(env.ActiveDirectoryEndpoint, test.config.TenantID)
			if err != nil {
				t.Fatalf("expected err to be nil, got: %v", err)
			}
			spt, err := adal.NewServicePrincipalToken(*oauthConfig, test.config.ClientID, test.config.ClientSecret, "https://vault.azure.net")
			if err != nil {
				t.Fatalf("expected err to be nil, got: %v", err)
			}
			if !reflect.DeepEqual(token, spt) {
				t.Fatalf("expected: %+v, got: %+v", spt, token)
			}
		})
	}
}
