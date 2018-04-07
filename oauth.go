// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package main

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"golang.org/x/crypto/pkcs12"
	"github.com/golang/glog"
	"github.com/ghodss/yaml"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
)

const (
)

var (
	oauthConfig 			*adal.OAuthConfig
)

// OAuthGrantType specifies which grant type to use.
type OAuthGrantType int

const (
	// OAuthGrantTypeServicePrincipal for client credentials flow
	OAuthGrantTypeServicePrincipal OAuthGrantType = iota
	// OAuthGrantTypeDeviceFlow for device-auth flow
	OAuthGrantTypeDeviceFlow
)

// AzureAuthConfig holds auth related part of cloud config
type AzureAuthConfig struct {
	// The cloud environment identifier. Takes values from https://github.com/Azure/go-autorest/blob/ec5f4903f77ed9927ac95b19ab8e44ada64c1356/autorest/azure/environments.go#L13
	Cloud string `json:"cloud"`
	// The AAD Tenant ID for the Subscription that the cluster is deployed in
	TenantID string `json:"tenantId"`
	// The ClientID for an AAD application with RBAC access to talk to Azure RM APIs
	AADClientID string `json:"aadClientId"`
	// The ClientSecret for an AAD application with RBAC access to talk to Azure RM APIs
	AADClientSecret string `json:"aadClientSecret"`
	// The path of a client certificate for an AAD application with RBAC access to talk to Azure RM APIs
	AADClientCertPath string `json:"aadClientCertPath"`
	// The password of the client certificate for an AAD application with RBAC access to talk to Azure RM APIs
	AADClientCertPassword string `json:"aadClientCertPassword"`
	// Use managed service identity for the virtual machine to access Azure ARM APIs
	UseManagedIdentityExtension bool `json:"useManagedIdentityExtension"`
	// The ID of the Azure Subscription that the cluster is deployed in
	SubscriptionID string `json:"subscriptionId"`
}

// Config holds the configuration parsed from the --cloud-config flag
// All fields are required unless otherwise specified
type Config struct {
	AzureAuthConfig
	// Resource Group for cluster
	ResourceGroup string `json:"resourceGroup"`
	// The kms provider vault name
	ProviderVaultName string `json:"providerVaultName"`
	// The kms provider key name
	ProviderKeyName string `json:"providerKeyName"`
	// The kms provider key version
	ProviderKeyVersion string `json:"providerKeyVersion"`
}

func AuthGrantType() OAuthGrantType {
	return OAuthGrantTypeServicePrincipal
}

func GetKMSProvider(configFilePath string) (vaultName *string, keyName *string, keyVersion *string, err error) {
	var configReader io.Reader
	var config Config

	if configFilePath != "" {
		var configFile *os.File
		configFile, err = os.Open(configFilePath)
		if err != nil {
			glog.Fatalf("Couldn't open cloud provider configuration %s: %#v",
				configFilePath, err)
			return nil, nil, nil, err
		}

		defer configFile.Close()
		configReader = configFile
		configContents, err := ioutil.ReadAll(configReader)
		if err != nil {
			return nil, nil, nil, err
		}
		err = yaml.Unmarshal(configContents, &config)
		if err != nil {
			return nil, nil, nil, err
		}
		_, err = ParseAzureEnvironment(config.Cloud)
		if err != nil {
			return nil, nil, nil, err
		}
		
		if (config.ProviderVaultName != "" ) {
			vaultName = &config.ProviderVaultName
		} else {
			return nil, nil, nil, fmt.Errorf("Unable to find KMS providerVaultName in configs")
		}
		if (config.ProviderKeyName != "" ) {
			keyName = &config.ProviderKeyName
		} else {
			return nil, nil, nil, fmt.Errorf("Unable to find KMS providerKeyName in configs")
		}
		keyVersion = &config.ProviderKeyVersion
		return vaultName, keyName, keyVersion, nil
	}
	return nil, nil, nil, fmt.Errorf("Cloud provider configuration file is missing")
}
func GetManagementToken(grantType OAuthGrantType, configFilePath string) (authorizer autorest.Authorizer, err error) {
	var configReader io.Reader
	var config Config
	
	if configFilePath != "" {
		var configFile *os.File
		configFile, err = os.Open(configFilePath)
		if err != nil {
			glog.Fatalf("Couldn't open cloud provider configuration %s: %#v",
				configFilePath, err)
			return nil, err
		}

		defer configFile.Close()
		configReader = configFile
		configContents, err := ioutil.ReadAll(configReader)
		if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(configContents, &config)
		if err != nil {
			return nil, err
		}
		env, err := ParseAzureEnvironment(config.Cloud)
		if err != nil {
			return nil, err
		}

		rmEndPoint := env.ResourceManagerEndpoint
		servicePrincipalToken, err := GetServicePrincipalToken(&config.AzureAuthConfig, env, rmEndPoint)
		if err != nil {
			return nil, err
		}
		authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)
		return authorizer, nil
	} 
	return
}

func GetKeyvaultToken(grantType OAuthGrantType, configFilePath string) (authorizer autorest.Authorizer, err error) {
	var configReader io.Reader
	var config Config
	
	if configFilePath != "" {
		var configFile *os.File
		configFile, err = os.Open(configFilePath)
		if err != nil {
			glog.Fatalf("Couldn't open cloud provider configuration %s: %#v",
				configFilePath, err)
			return nil, err
		}

		defer configFile.Close()
		configReader = configFile
		configContents, err := ioutil.ReadAll(configReader)
		if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(configContents, &config)
		if err != nil {
			return nil, err
		}
		env, err := ParseAzureEnvironment(config.Cloud)
		if err != nil {
			return nil, err
		}

		kvEndPoint := env.KeyVaultEndpoint
		if '/' == kvEndPoint[len(kvEndPoint)-1] {
			kvEndPoint = kvEndPoint[:len(kvEndPoint)-1]
		}
		servicePrincipalToken, err := GetServicePrincipalToken(&config.AzureAuthConfig, env, kvEndPoint)
		if err != nil {
			return nil, err
		}
		authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)
		return authorizer, nil
	} 
	return
}
func GetAzureAuthConfig(configFilePath string) (azConfig *AzureAuthConfig, err error) {
	var configReader io.Reader
	var config Config

	if configFilePath != "" {
		var configFile *os.File
		configFile, err = os.Open(configFilePath)
		if err != nil {
			glog.Errorf("Couldn't open cloud provider configuration %s: %#v",
				configFilePath, err)
			return nil, err
		}

		defer configFile.Close()
		configReader = configFile
		configContents, err := ioutil.ReadAll(configReader)
		if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(configContents, &config)
		if err != nil {
			return nil, err
		}
		_, err = ParseAzureEnvironment(config.Cloud)
		if err != nil {
			return nil, err
		}
		
		if ( &config.AzureAuthConfig != nil ) {
			return &config.AzureAuthConfig, nil
		}
		return nil, fmt.Errorf("Cloud provider configuration file is missing AzureAuthConfig")
	}
	return nil, fmt.Errorf("Cloud provider configuration file is missing")
}

func GetResourceGroup(configFilePath string) (resourceGroup *string, err error) {
	var configReader io.Reader
	var config Config

	if configFilePath != "" {
		var configFile *os.File
		configFile, err = os.Open(configFilePath)
		if err != nil {
			glog.Fatalf("Couldn't open cloud provider configuration %s: %#v",
				configFilePath, err)
			return nil, err
		}

		defer configFile.Close()
		configReader = configFile
		configContents, err := ioutil.ReadAll(configReader)
		if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(configContents, &config)
		if err != nil {
			return nil, err
		}
		_, err = ParseAzureEnvironment(config.Cloud)
		if err != nil {
			return nil, err
		}
		
		if ( config.ResourceGroup != "" ) {
			resourceGroup = &config.ResourceGroup
			return resourceGroup, nil
		}
		return nil, fmt.Errorf("Cloud provider configuration file is missing resourceGroup")
	}
	return nil, fmt.Errorf("Cloud provider configuration file is missing")
}

// GetServicePrincipalToken creates a new service principal token based on the configuration
func GetServicePrincipalToken(config *AzureAuthConfig, env *azure.Environment, resource string) (*adal.ServicePrincipalToken, error) {
	oauthConfig, err := adal.NewOAuthConfig(env.ActiveDirectoryEndpoint, config.TenantID)
	if err != nil {
		return nil, fmt.Errorf("creating the OAuth config: %v", err)
	}

	if config.UseManagedIdentityExtension {
		glog.V(2).Infoln("azure: using managed identity extension to retrieve access token")
		msiEndpoint, err := adal.GetMSIVMEndpoint()
		if err != nil {
			return nil, fmt.Errorf("Getting the managed service identity endpoint: %v", err)
		}
		return adal.NewServicePrincipalTokenFromMSI(
			msiEndpoint,
			resource)
	}

	if len(config.AADClientSecret) > 0 {
		glog.V(2).Infoln("azure: using client_id+client_secret to retrieve access token")
		return adal.NewServicePrincipalToken(
			*oauthConfig,
			config.AADClientID,
			config.AADClientSecret,
			resource)
	}

	if len(config.AADClientCertPath) > 0 && len(config.AADClientCertPassword) > 0 {
		glog.V(2).Infoln("azure: using jwt client_assertion (client_cert+client_private_key) to retrieve access token")
		certData, err := ioutil.ReadFile(config.AADClientCertPath)
		if err != nil {
			return nil, fmt.Errorf("reading the client certificate from file %s: %v", config.AADClientCertPath, err)
		}
		certificate, privateKey, err := decodePkcs12(certData, config.AADClientCertPassword)
		if err != nil {
			return nil, fmt.Errorf("decoding the client certificate: %v", err)
		}
		return adal.NewServicePrincipalTokenFromCertificate(
			*oauthConfig,
			config.AADClientID,
			certificate,
			privateKey,
			env.ServiceManagementEndpoint)
	}

	return nil, fmt.Errorf("No credentials provided for AAD application %s", config.AADClientID)
}

// ParseAzureEnvironment returns azure environment by name
func ParseAzureEnvironment(cloudName string) (*azure.Environment, error) {
	var env azure.Environment
	var err error
	if cloudName == "" {
		env = azure.PublicCloud
	} else {
		env, err = azure.EnvironmentFromName(cloudName)
	}
	return &env, err
}

// decodePkcs12 decodes a PKCS#12 client certificate by extracting the public certificate and
// the private RSA key
func decodePkcs12(pkcs []byte, password string) (*x509.Certificate, *rsa.PrivateKey, error) {
	privateKey, certificate, err := pkcs12.Decode(pkcs, password)
	if err != nil {
		return nil, nil, fmt.Errorf("decoding the PKCS#12 client certificate: %v", err)
	}
	rsaPrivateKey, isRsaKey := privateKey.(*rsa.PrivateKey)
	if !isRsaKey {
		return nil, nil, fmt.Errorf("PKCS#12 certificate must contain a RSA private key")
	}

	return certificate, rsaPrivateKey, nil
}
