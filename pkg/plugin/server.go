// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"

	"github.com/Azure/kubernetes-kms/pkg/config"
	"github.com/Azure/kubernetes-kms/pkg/version"

	k8spb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
	"k8s.io/klog/v2"
)

// KeyManagementServiceServer is a gRPC server.
type KeyManagementServiceServer struct {
	kvClient Client
}

// New creates an instance of the KMS Service Server.
func New(ctx context.Context, configFilePath, vaultName, keyName, keyVersion, vaultSKU string) (*KeyManagementServiceServer, error) {
	cfg, err := config.GetAzureConfig(configFilePath)
	if err != nil {
		return nil, err
	}
	kvClient, err := newKeyVaultClient(cfg, vaultName, keyName, keyVersion, vaultSKU)
	if err != nil {
		return nil, err
	}
	err = kvClient.CheckIfKeyExists(ctx)
	if err != nil {
		return nil, err
	}
	return &KeyManagementServiceServer{
		kvClient: kvClient,
	}, nil
}

func (s *KeyManagementServiceServer) Version(ctx context.Context, request *k8spb.VersionRequest) (*k8spb.VersionResponse, error) {
	return &k8spb.VersionResponse{
		Version:        version.APIVersion,
		RuntimeName:    version.Runtime,
		RuntimeVersion: version.BuildVersion,
	}, nil
}

func (s *KeyManagementServiceServer) Encrypt(ctx context.Context, request *k8spb.EncryptRequest) (*k8spb.EncryptResponse, error) {
	cipher, err := s.kvClient.Encrypt(ctx, request.Plain)
	if err != nil {
		klog.ErrorS(err, "failed to encrypt")
		return &k8spb.EncryptResponse{}, err
	}
	klog.Infof("encrypt request complete")
	return &k8spb.EncryptResponse{Cipher: cipher}, nil
}

func (s *KeyManagementServiceServer) Decrypt(ctx context.Context, request *k8spb.DecryptRequest) (*k8spb.DecryptResponse, error) {
	plain, err := s.kvClient.Decrypt(ctx, request.Cipher)
	if err != nil {
		klog.ErrorS(err, "failed to decrypt")
		return &k8spb.DecryptResponse{}, err
	}
	klog.Infof("decrypt request complete")
	return &k8spb.DecryptResponse{Plain: plain}, nil
}
