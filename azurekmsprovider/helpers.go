// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package main

import (
	"flag"
	"os"
	"strings"

	"github.com/subosito/gotenv"
)

var (
	keepResources     bool
	deviceFlow        bool
)

// ParseDeviceFlow parses the auth grant type to be used
// The caller should do flag.Parse()
func ParseDeviceFlow() error {
	err := LoadEnvVars()
	if err != nil {
		return err
	}

	if os.Getenv("AZ_AUTH_DEVICEFLOW") != "" {
		deviceFlow = true
	}
	flag.BoolVar(&deviceFlow, "deviceFlow", deviceFlow, "Use device flow for authentication. This flag should be used with -v flag. Default authentication is service principal.")
	return nil
}

// getters

// KeepResources indicates whether resources created by samples should be retained.
func KeepResources() bool {
	return keepResources
}

// GroupPrefix specifies the prefix sample resource groups should have
func GroupPrefix() string {
	return "group-azure-samples-go"
}

// DeviceFlow returns if device flow has been set as auth grant type
func DeviceFlow() bool {
	return deviceFlow
}

// LoadEnvVars loads environment variables.
func LoadEnvVars() error {
	err := gotenv.Load() // to allow use of .env file
	if err != nil && !strings.HasPrefix(err.Error(), "open .env:") {
		return err
	}
	return nil
}