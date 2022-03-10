// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package consts

const (
	// In proxy mode, the header is added into the requests from kms-plugin.
	// The proxy will check the header and forward the request to different destinations.
	// e.g. When the value of the header "x-azure-proxy-target" is "KeyVault", the request
	// is forwared to Azure Key Vault by the proxy.
	RequestHeaderTargetType        = "x-azure-proxy-target"
	TargetTypeAzureActiveDirectory = "AzureActiveDirectory"
	TargetTypeKeyVault             = "KeyVault"
)
