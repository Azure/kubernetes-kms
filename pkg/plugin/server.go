// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"time"

	"github.com/Azure/kubernetes-kms/pkg/config"
	"github.com/Azure/kubernetes-kms/pkg/metrics"
	"github.com/Azure/kubernetes-kms/pkg/version"

	k8spb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
	"k8s.io/klog/v2"
)

// KeyManagementServiceServer is a gRPC server.
type KeyManagementServiceServer struct {
	kvClient Client
	reporter metrics.StatsReporter
}

// New creates an instance of the KMS Service Server.
func New(ctx context.Context, configFilePath, vaultName, keyName, keyVersion string, proxyMode bool, proxyAddress string, proxyPort int) (*KeyManagementServiceServer, error) {
	cfg, err := config.GetAzureConfig(configFilePath)
	if err != nil {
		return nil, err
	}
	kvClient, err := newKeyVaultClient(cfg, vaultName, keyName, keyVersion, proxyMode, proxyAddress, proxyPort)
	if err != nil {
		return nil, err
	}
	return &KeyManagementServiceServer{
		kvClient: kvClient,
		reporter: metrics.NewStatsReporter(),
	}, nil
}

// Version of kms
func (s *KeyManagementServiceServer) Version(ctx context.Context, request *k8spb.VersionRequest) (*k8spb.VersionResponse, error) {
	return &k8spb.VersionResponse{
		Version:        version.APIVersion,
		RuntimeName:    version.Runtime,
		RuntimeVersion: version.BuildVersion,
	}, nil
}

// Encrypt message
func (s *KeyManagementServiceServer) Encrypt(ctx context.Context, request *k8spb.EncryptRequest) (*k8spb.EncryptResponse, error) {
	start := time.Now()

	var err error
	defer func() {
		errors := ""
		status := metrics.SuccessStatusTypeValue
		if err != nil {
			status = metrics.ErrorStatusTypeValue
			errors = err.Error()
		}
		s.reporter.ReportRequest(ctx, metrics.EncryptOperationTypeValue, status, time.Since(start).Seconds(), errors)
	}()

	klog.V(2).Infof("encrypt request started")
	cipher, err := s.kvClient.Encrypt(ctx, request.Plain)
	if err != nil {
		klog.ErrorS(err, "failed to encrypt")
		return &k8spb.EncryptResponse{}, err
	}
	klog.V(2).Infof("encrypt request complete")
	return &k8spb.EncryptResponse{Cipher: cipher}, nil
}

// Decrypt message
func (s *KeyManagementServiceServer) Decrypt(ctx context.Context, request *k8spb.DecryptRequest) (*k8spb.DecryptResponse, error) {
	start := time.Now()

	var err error
	defer func() {
		errors := ""
		status := metrics.SuccessStatusTypeValue
		if err != nil {
			status = metrics.ErrorStatusTypeValue
			errors = err.Error()
		}
		s.reporter.ReportRequest(ctx, metrics.DecryptOperationTypeValue, status, time.Since(start).Seconds(), errors)
	}()

	klog.V(2).Infof("decrypt request started")
	plain, err := s.kvClient.Decrypt(ctx, request.Cipher)
	if err != nil {
		klog.ErrorS(err, "failed to decrypt")
		return &k8spb.DecryptResponse{}, err
	}
	klog.V(2).Infof("decrypt request complete")
	return &k8spb.DecryptResponse{Plain: plain}, nil
}
