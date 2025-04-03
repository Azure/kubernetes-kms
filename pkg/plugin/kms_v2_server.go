// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys"
	"github.com/Azure/kubernetes-kms/pkg/metrics"
	"github.com/Azure/kubernetes-kms/pkg/version"

	kmsv2 "k8s.io/kms/apis/v2"
	"monis.app/mlog"
)

// KeyManagementServiceV2Server is a gRPC server.
type KeyManagementServiceV2Server struct {
	kvClient            Client
	reporter            metrics.StatsReporter
	encryptionAlgorithm azkeys.EncryptionAlgorithm
}

// NewKMSv2Server creates an instance of the KMS Service Server with v2 apis.
func NewKMSv2Server(kvClient Client) (*KeyManagementServiceV2Server, error) {
	statsReporter, err := metrics.NewStatsReporter()
	if err != nil {
		return nil, fmt.Errorf("failed to create stats reporter: %w", err)
	}

	return &KeyManagementServiceV2Server{
		kvClient:            kvClient,
		reporter:            statsReporter,
		encryptionAlgorithm: azkeys.EncryptionAlgorithmRSAOAEP256,
	}, nil
}

// Status returns the health status of the KMS plugin.
func (s *KeyManagementServiceV2Server) Status(ctx context.Context, _ *kmsv2.StatusRequest) (*kmsv2.StatusResponse, error) {
	// We perform a simple encrypt/decrypt operation to verify the plugin's connectivity with Key Vault.
	// The KMS invokes the Status API every minute, resulting in 120 calls per hour to the Key Vault.
	// This volume of calls is well within the permissible limit of Key Vault.
	encryptResponse, err := s.kvClient.Encrypt(ctx, []byte(healthCheckPlainText), s.encryptionAlgorithm)
	if err != nil {
		mlog.Error("failed to encrypt healthcheck call", err)
		return nil, err
	}

	decryptedText, err := s.kvClient.Decrypt(
		ctx,
		encryptResponse.Ciphertext,
		s.encryptionAlgorithm,
		version.KMSv2APIVersion,
		encryptResponse.Annotations,
		encryptResponse.KeyID,
	)
	if err != nil {
		mlog.Error("failed to decrypt healthcheck call", err)
		return nil, err
	}

	if string(decryptedText) != healthCheckPlainText {
		err = fmt.Errorf("decrypted text does not match")
		mlog.Error("healthcheck failed", err)
		return nil, err
	}

	return &kmsv2.StatusResponse{
		Version: version.KMSv2APIVersion,
		Healthz: "ok",
		KeyId:   encryptResponse.KeyID,
	}, nil
}

// Encrypt message.
func (s *KeyManagementServiceV2Server) Encrypt(ctx context.Context, request *kmsv2.EncryptRequest) (*kmsv2.EncryptResponse, error) {
	mlog.Debug("encrypt request received", "uid", request.Uid)
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

	mlog.Info("encrypt request started", "uid", request.Uid)
	encryptResponse, err := s.kvClient.Encrypt(ctx, request.Plaintext, s.encryptionAlgorithm)
	if err != nil {
		mlog.Error("failed to encrypt", err, "uid", request.Uid)
		return &kmsv2.EncryptResponse{}, err
	}
	mlog.Info("encrypt request complete", "uid", request.Uid)

	return &kmsv2.EncryptResponse{
		Ciphertext:  encryptResponse.Ciphertext,
		KeyId:       encryptResponse.KeyID,
		Annotations: encryptResponse.Annotations,
	}, nil
}

// Decrypt message.
func (s *KeyManagementServiceV2Server) Decrypt(ctx context.Context, request *kmsv2.DecryptRequest) (*kmsv2.DecryptResponse, error) {
	mlog.Debug("decrypt request received", "uid", request.Uid)
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

	mlog.Info("decrypt request started", "uid", request.Uid)

	plainText, err := s.kvClient.Decrypt(
		ctx,
		request.Ciphertext,
		s.encryptionAlgorithm,
		version.KMSv2APIVersion,
		request.Annotations,
		request.KeyId,
	)
	if err != nil {
		mlog.Error("failed to decrypt", err, "uid", request.Uid)
		return &kmsv2.DecryptResponse{}, err
	}
	mlog.Info("decrypt request complete", "uid", request.Uid)

	return &kmsv2.DecryptResponse{
		Plaintext: plainText,
	}, nil
}
