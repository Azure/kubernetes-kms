// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"regexp"

	"github.com/Azure/kubernetes-kms/pkg/config"
	"github.com/Azure/kubernetes-kms/pkg/consts"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"golang.org/x/crypto/pkcs12"
	"k8s.io/klog/v2"
)

// GetTokenCredential returns token credential
func GetTokenCredential(config *config.AzureConfig, aadEndpoint, resource string, proxyMode bool) (cred azcore.TokenCredential, err error) {
	return getCredential(config, aadEndpoint, resource, proxyMode)
}

// getCredential returns a token provider for the specified resource
func getCredential(config *config.AzureConfig, aadEndpoint, resource string, proxyMode bool) (azcore.TokenCredential, error) {
	if config.UseManagedIdentityExtension {
		klog.V(2).InfoS("using managed identity to retrieve access token", "clientID", redactClientCredentials(config.UserAssignedIdentityID))
		opts := &azidentity.ManagedIdentityCredentialOptions{
			ID: azidentity.ClientID(config.UserAssignedIdentityID),
		}
		return azidentity.NewManagedIdentityCredential(opts)
	}

	if len(config.ClientSecret) > 0 && len(config.ClientID) > 0 {
		klog.V(2).InfoS("using client_id+client_secret to retrieve access token",
			"clientID", redactClientCredentials(config.ClientID), "clientSecret", redactClientCredentials(config.ClientSecret))

		opts := &azidentity.ClientSecretCredentialOptions{
			ClientOptions: azcore.ClientOptions{
				Cloud: cloud.Configuration{
					ActiveDirectoryAuthorityHost: aadEndpoint,
				},
			},
		}

		if proxyMode {
			opts.ClientOptions.Transport = &transporter{}
		}
		return azidentity.NewClientSecretCredential(config.TenantID, config.ClientID, config.ClientSecret, opts)
	}

	if len(config.AADClientCertPath) > 0 && len(config.AADClientCertPassword) > 0 {
		klog.V(2).Info("using jwt client_assertion (client_cert+client_private_key) to retrieve access token")
		certData, err := os.ReadFile(config.AADClientCertPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read client certificate from file %s, error: %w", config.AADClientCertPath, err)
		}
		certificate, privateKey, err := decodePkcs12(certData, config.AADClientCertPassword)
		if err != nil {
			return nil, fmt.Errorf("failed to decode the client certificate, error: %v", err)
		}

		opts := &azidentity.ClientCertificateCredentialOptions{
			ClientOptions: azcore.ClientOptions{
				Cloud: cloud.Configuration{
					ActiveDirectoryAuthorityHost: aadEndpoint,
				},
			},
		}

		if proxyMode {
			opts.ClientOptions.Transport = &transporter{}
		}

		return azidentity.NewClientCertificateCredential(config.TenantID, config.ClientID, []*x509.Certificate{certificate}, privateKey, opts)
	}

	return nil, fmt.Errorf("no credentials provided for accessing keyvault")
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

// redactClientCredentials applies regex to a sensitive string and return the redacted value
func redactClientCredentials(sensitiveString string) string {
	r, _ := regexp.Compile(`^(\S{4})(\S|\s)*(\S{4})$`)
	return r.ReplaceAllString(sensitiveString, "$1##### REDACTED #####$3")
}

type transporter struct {
}

func (t *transporter) Do(req *http.Request) (*http.Response, error) {
	// adds the target header if proxy mode is enabled
	req.Header.Set(consts.RequestHeaderTargetType, consts.TargetTypeAzureActiveDirectory)
	return http.DefaultClient.Do(req)
}
