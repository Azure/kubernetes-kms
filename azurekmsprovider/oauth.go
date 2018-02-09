// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package main

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
)

const (
	samplesAppID = "bee3737f-b06f-444f-b3c3-5b0f3fce46ea"
)

var (
	// for service principal and device
	clientID    			string
	oauthConfig 			*adal.OAuthConfig
	armToken    			adal.OAuthTokenProvider
	graphToken  			adal.OAuthTokenProvider
	resourceGroupName       string
	subscriptionID   		string
	tenantID       			string
	clientSecret   			string
)

// OAuthGrantType specifies which grant type to use.
type OAuthGrantType int

const (
	// OAuthGrantTypeServicePrincipal for client credentials flow
	OAuthGrantTypeServicePrincipal OAuthGrantType = iota
	// OAuthGrantTypeDeviceFlow for device-auth flow
	OAuthGrantTypeDeviceFlow
)

func init() {
	err := parseArgs()
	if err != nil {
		log.Fatalf("failed to parse args: %s\n", err)
	}
}

func parseArgs() error {
	err := LoadEnvVars()
	if err != nil {
		return err
	}

	tenantID = os.Getenv("AZ_TENANT_ID")
	if tenantID == "" {
		log.Println("tenant id missing")
	}
	clientID = os.Getenv("AZ_CLIENT_ID")
	if clientID == "" {
		log.Println("client id missing")
	}
	clientSecret = os.Getenv("AZ_CLIENT_SECRET")
	if clientSecret == "" {
		log.Println("client secret missing")
	}
	resourceGroupName = os.Getenv("AZ_RESOURCE_GROUP_NAME")
	if resourceGroupName == "" {
		log.Println("resourcegroup Name missing")
	}
	subscriptionID = os.Getenv("AZ_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		log.Println("subscription id missing")
	}


	if !(len(tenantID) > 0) || !(len(clientID) > 0) || !(len(clientSecret) > 0) {
		return errors.New("tenant id, client id, and client secret must be specified via env var or flags")
	}

	oauthConfig, err = adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, tenantID)

	return err
}

// ClientID gets the client ID
func ClientID() string {
	return clientID
}

// TenantID gets the client ID
func TenantID() string {
	return tenantID
}

// ClientSecret gets the client secret
func ClientSecret() string {
	return clientSecret
}

func AuthGrantType() OAuthGrantType {
	return OAuthGrantTypeServicePrincipal
}

// SubscriptionID returns the ID of the subscription to use.
func SubscriptionID() string {
	fmt.Println("subscriptionID:")
	fmt.Println(subscriptionID)
	return subscriptionID
}

// ResourceGroupName returns the name of the resource group to use.
func ResourceGroupName() string {
	fmt.Println("ResourceGroupName:")
	fmt.Println(resourceGroupName)
	return resourceGroupName
}

// GetResourceManagementToken gets an OAuth token for managing resources using the specified grant type.
func GetResourceManagementToken(grantType OAuthGrantType) (adal.OAuthTokenProvider, error) {
	if armToken != nil {
		return armToken, nil
	}

	token, err := getToken(grantType, azure.PublicCloud.ResourceManagerEndpoint)
	if err == nil {
		armToken = token
	}

	return token, err
}

func getToken(grantType OAuthGrantType, endpoint string) (token adal.OAuthTokenProvider, err error) {
	return getServicePrincipalToken(endpoint)
}

func getServicePrincipalToken(endpoint string) (adal.OAuthTokenProvider, error) {
	return adal.NewServicePrincipalToken(
		*oauthConfig,
		clientID,
		clientSecret,
		endpoint)
}

func GetKeyvaultToken(grantType OAuthGrantType) (authorizer autorest.Authorizer, err error) {
	fmt.Println(tenantID)
	fmt.Println(clientID)
	fmt.Println(clientSecret)
	config, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, tenantID)
	updatedAuthorizeEndpoint, err := url.Parse("https://login.windows.net/" + tenantID + "/oauth2/token")
	config.AuthorizeEndpoint = *updatedAuthorizeEndpoint
	if err != nil {
		return
	}

	spt, err := adal.NewServicePrincipalToken(
		*config,
		clientID,
		clientSecret,
		"https://vault.azure.net")

	if err != nil {
		return authorizer, err
	}
	authorizer = autorest.NewBearerAuthorizer(spt)
	return authorizer, nil
}
