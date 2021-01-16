package config

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
	"k8s.io/klog/v2"
)

// AzureConfig is representing /etc/kubernetes/azure.json
type AzureConfig struct {
	Cloud                       string `json:"cloud" yaml:"cloud"`
	TenantID                    string `json:"tenantId" yaml:"tenantId"`
	ClientID                    string `json:"aadClientId" yaml:"aadClientId"`
	ClientSecret                string `json:"aadClientSecret" yaml:"aadClientSecret"`
	SubscriptionID              string `json:"subscriptionId" yaml:"subscriptionId"`
	ResourceGroupName           string `json:"resourceGroup" yaml:"resourceGroup"`
	SecurityGroupName           string `json:"securityGroupName" yaml:"securityGroupName"`
	VMType                      string `json:"vmType" yaml:"vmType"`
	UseManagedIdentityExtension bool   `json:"useManagedIdentityExtension,omitempty" yaml:"useManagedIdentityExtension,omitempty"`
	UserAssignedIdentityID      string `json:"userAssignedIdentityID,omitempty" yaml:"userAssignedIdentityID,omitempty"`
	AADClientCertPath           string `json:"aadClientCertPath" yaml:"aadClientCertPath"`
	AADClientCertPassword       string `json:"aadClientCertPassword" yaml:"aadClientCertPassword"`
}

// GetAzureConfig returns configs in the azure.json cloud provider file
func GetAzureConfig(configFile string) (config *AzureConfig, err error) {
	cfg := AzureConfig{}

	klog.V(5).Infof("populating AzureConfig from %s", configFile)
	bytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config file %s, error: %+v", configFile, err)
	}
	if err = yaml.Unmarshal(bytes, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal azure.json, error: %+v", err)
	}
	return &cfg, nil
}
