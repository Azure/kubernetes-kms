package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
	"monis.app/mlog"
)

// AzureConfig is representing /etc/kubernetes/azure.json.
type AzureConfig struct {
	Cloud                       string `json:"cloud" yaml:"cloud"`
	TenantID                    string `json:"tenantId" yaml:"tenantId"`
	ClientID                    string `json:"aadClientId" yaml:"aadClientId"`
	ClientSecret                string `json:"aadClientSecret" yaml:"aadClientSecret"`
	UseManagedIdentityExtension bool   `json:"useManagedIdentityExtension,omitempty" yaml:"useManagedIdentityExtension,omitempty"`
	UserAssignedIdentityID      string `json:"userAssignedIdentityID,omitempty" yaml:"userAssignedIdentityID,omitempty"`
	AADClientCertPath           string `json:"aadClientCertPath" yaml:"aadClientCertPath"`
	AADClientCertPassword       string `json:"aadClientCertPassword" yaml:"aadClientCertPassword"`
}

// GetAzureConfig returns configs in the azure.json cloud provider file.
func GetAzureConfig(configFile string) (config *AzureConfig, err error) {
	cfg := AzureConfig{}

	mlog.Trace("populating AzureConfig from config file", "configFile", configFile)
	bytes, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config file %s, error: %+v", configFile, err)
	}
	if err = yaml.Unmarshal(bytes, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal azure.json, error: %+v", err)
	}
	return &cfg, nil
}
