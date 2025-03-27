// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package auth

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/Azure/kubernetes-kms/pkg/config"
	"github.com/Azure/kubernetes-kms/pkg/consts"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/msi-dataplane/pkg/dataplane"
	"github.com/jongio/azidext/go/azidext"
	"golang.org/x/crypto/pkcs12"
	"monis.app/mlog"
)

// GetKeyvaultToken returns token for Keyvault endpoint.
func GetKeyvaultToken(ctx context.Context, config *config.AzureConfig, env *azure.Environment, resource string, proxyMode bool) (authorizer autorest.Authorizer, err error) {
	servicePrincipalToken, cred, err := GetServicePrincipalToken(ctx, config, env.ActiveDirectoryEndpoint, resource, proxyMode)
	if err != nil {
		return nil, err
	}
	if cred != nil {
		authorizer = azidext.NewTokenCredentialAdapter(cred, []string{"https://management.azure.com//.default"})
	} else {
		authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)
	}
	return authorizer, nil
}

// GetServicePrincipalToken creates a new service principal token based on the configuration.
func GetServicePrincipalToken(ctx context.Context, config *config.AzureConfig, aadEndpoint, resource string, proxyMode bool) (adal.OAuthTokenProvider, azcore.TokenCredential, error) {
	oauthConfig, err := adal.NewOAuthConfig(aadEndpoint, config.TenantID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create OAuth config, error: %v", err)
	}

	if config.UseManagedIdentityExtension {
		mlog.Info("using managed identity extension to retrieve access token")
		msiEndpoint, err := adal.GetMSIVMEndpoint()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get managed service identity endpoint, error: %v", err)
		}
		// using user-assigned managed identity to access keyvault
		if len(config.UserAssignedIdentityID) > 0 {
			mlog.Info("using User-assigned managed identity to retrieve access token", "clientID", redactClientCredentials(config.UserAssignedIdentityID))
			spt, err := adal.NewServicePrincipalTokenFromMSIWithUserAssignedID(msiEndpoint,
				resource,
				config.UserAssignedIdentityID)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to create service principal token, error: %v", err)
			}
			return spt, nil, nil
		}
		mlog.Info("using system-assigned managed identity to retrieve access token")
		// using system-assigned managed identity to access keyvault
		spt, err := adal.NewServicePrincipalTokenFromMSI(
			msiEndpoint,
			resource)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create service principal token, error: %v", err)
		}
		return spt, nil, nil
	}

	if len(config.ClientSecret) > 0 && len(config.ClientID) > 0 {
		mlog.Info("azure: using client_id+client_secret to retrieve access token",
			"clientID", redactClientCredentials(config.ClientID), "clientSecret", redactClientCredentials(config.ClientSecret))

		spt, err := adal.NewServicePrincipalToken(
			*oauthConfig,
			config.ClientID,
			config.ClientSecret,
			resource)
		if err != nil {
			return nil, nil, err
		}
		if proxyMode {
			return addTargetTypeHeader(spt), nil, nil
		}
		return spt, nil, nil
	}

	if len(config.AADClientCertPath) > 0 && len(config.AADClientCertPassword) > 0 {
		mlog.Info("using jwt client_assertion (client_cert+client_private_key) to retrieve access token")
		certData, err := os.ReadFile(config.AADClientCertPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read client certificate from file %s, error: %v", config.AADClientCertPath, err)
		}
		certificate, privateKey, err := decodePkcs12(certData, config.AADClientCertPassword)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to decode the client certificate, error: %v", err)
		}
		spt, err := adal.NewServicePrincipalTokenFromCertificate(
			*oauthConfig,
			config.ClientID,
			certificate,
			privateKey,
			resource)
		if err != nil {
			return nil, nil, err
		}
		if proxyMode {
			return addTargetTypeHeader(spt), nil, nil
		}
		return spt, nil, nil
	}

	if len(config.AADMSIDataPlaneIdentityPath) > 0 {
		mlog.Info("using MSI Data Plane Identity to retrieve access token")
		cloudType := parseCloudType(config.Cloud)
		options := azcore.ClientOptions{
			Cloud: cloudType,
		}
		cred, err := dataplane.NewUserAssignedIdentityCredential(ctx, config.AADMSIDataPlaneIdentityPath, dataplane.WithClientOpts(options))
		if err != nil {
			return nil, nil, err
		}

		return nil, cred, nil
	}

	return nil, nil, fmt.Errorf("no credentials provided for accessing keyvault")
}

func parseCloudType(cloudType string) cloud.Configuration {
	cloudType = strings.ToUpper(cloudType)
	switch cloudType {
	case "AZURECLOUD":
		return cloud.AzurePublic
	case "AZUREPUBLICCLOUD":
		return cloud.AzurePublic
	case "PUBLIC":
		return cloud.AzurePublic
	case "AZURECHINACLOUD":
		return cloud.AzureChina
	case "CHINA":
		return cloud.AzureChina
	case "AZUREUSGOVERNMENT":
		return cloud.AzureGovernment
	case "AZUREUSGOVERNMENTCLOUD":
		return cloud.AzureGovernment
	case "USGOVERNMENT":
		return cloud.AzureGovernment
	default:
		return cloud.AzurePublic
	}
}

// ParseAzureEnvironment returns azure environment by name.
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
// the private RSA key.
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

// redactClientCredentials applies regex to a sensitive string and return the redacted value.
func redactClientCredentials(sensitiveString string) string {
	r := regexp.MustCompile(`^(\S{4})(\S|\s)*(\S{4})$`)
	return r.ReplaceAllString(sensitiveString, "$1##### REDACTED #####$3")
}

// addTargetTypeHeader adds the target header if proxy mode is enabled.
func addTargetTypeHeader(spt *adal.ServicePrincipalToken) *adal.ServicePrincipalToken {
	spt.SetSender(autorest.CreateSender(
		(func() autorest.SendDecorator {
			return func(s autorest.Sender) autorest.Sender {
				return autorest.SenderFunc(func(r *http.Request) (*http.Response, error) {
					r.Header.Set(consts.RequestHeaderTargetType, consts.TargetTypeAzureActiveDirectory)
					return s.Do(r)
				})
			}
		})()))
	return spt
}
