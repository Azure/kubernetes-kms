// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package auth

import (
	"strings"
	"testing"
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
