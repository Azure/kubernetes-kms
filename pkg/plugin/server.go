// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/kubernetes-kms/pkg/metrics"
	"github.com/Azure/kubernetes-kms/pkg/version"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"k8s.io/klog/v2"
	kmsv1 "k8s.io/kms/apis/v1beta1"
)

// KeyManagementServiceServer is a gRPC server.
type KeyManagementServiceServer struct {
	kvClient            Client
	reporter            metrics.StatsReporter
	encryptionAlgorithm keyvault.JSONWebKeyEncryptionAlgorithm
}

// Config is the configuration for the KMS plugin.
type Config struct {
	ConfigFilePath string
	KeyVaultName   string
	KeyName        string
	KeyVersion     string
	ManagedHSM     bool
	ProxyMode      bool
	ProxyAddress   string
	ProxyPort      int
}

// NewKMSv1Server creates an instance of the KMS Service Server.
func NewKMSv1Server(kvClient Client) (*KeyManagementServiceServer, error) {
	statsReporter, err := metrics.NewStatsReporter()
	if err != nil {
		return nil, fmt.Errorf("failed to create stats reporter: %w", err)
	}

	return &KeyManagementServiceServer{
		kvClient:            kvClient,
		reporter:            statsReporter,
		encryptionAlgorithm: keyvault.RSA15,
	}, nil
}

// Version of kms.
func (s *KeyManagementServiceServer) Version(_ context.Context, _ *kmsv1.VersionRequest) (*kmsv1.VersionResponse, error) {
	return &kmsv1.VersionResponse{
		Version:        version.KMSv1APIVersion,
		RuntimeName:    version.Runtime,
		RuntimeVersion: version.BuildVersion,
	}, nil
}

// Encrypt message.
func (s *KeyManagementServiceServer) Encrypt(ctx context.Context, request *kmsv1.EncryptRequest) (*kmsv1.EncryptResponse, error) {
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

	klog.V(2).Info("encrypt request started")
	encryptResponse, err := s.kvClient.Encrypt(ctx, request.Plain, s.encryptionAlgorithm)
	if err != nil {
		klog.ErrorS(err, "failed to encrypt")
		return &kmsv1.EncryptResponse{}, err
	}
	klog.V(2).Info("encrypt request complete")
	return &kmsv1.EncryptResponse{
		Cipher: encryptResponse.Ciphertext,
	}, nil
}

// Decrypt message.
func (s *KeyManagementServiceServer) Decrypt(ctx context.Context, request *kmsv1.DecryptRequest) (*kmsv1.DecryptResponse, error) {
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

	klog.V(2).Info("decrypt request started")
	plain, err := s.kvClient.Decrypt(
		ctx,
		request.Cipher,
		s.encryptionAlgorithm,
		request.Version,
		nil,
		"",
	)
	if err != nil {
		klog.ErrorS(err, "failed to decrypt")
		return &kmsv1.DecryptResponse{}, err
	}
	klog.V(2).Info("decrypt request complete")
	return &kmsv1.DecryptResponse{Plain: plain}, nil
}
