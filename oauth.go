// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package main

import (
	"errors"
	"log"
	"net/url"
	"os"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
)

const (
)

var (
	// for service principal and device
	clientID    			string
	oauthConfig 			*adal.OAuthConfig
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

	oauthConfig, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, tenantID)
	if err != nil || oauthConfig == nil {
		log.Println("failed to get oauth config: error: %v", err)
	}
	return err
}

func AuthGrantType() OAuthGrantType {
	return OAuthGrantTypeServicePrincipal
}

func GetKeyvaultToken(grantType OAuthGrantType) (authorizer autorest.Authorizer, err error) {
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
