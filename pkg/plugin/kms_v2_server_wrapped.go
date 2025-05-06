// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/Azure/kubernetes-kms/pkg/metrics"
	"github.com/Azure/kubernetes-kms/pkg/plugin/aes"
	"github.com/Azure/kubernetes-kms/pkg/version"
	"monis.app/mlog"

	kmsv2 "k8s.io/kms/apis/v2"
	"k8s.io/kms/pkg/value"
)

const (
	staticKeyID                      = "1" // we can take this as input if we want to
	wrappedEncryptionResponseVersion = "2"
	authenticatedDataAnnotationKey   = "authenticated-data.azure.akv.io"
)

type KeyManagementServiceV2ServerWrapped struct {
	transformer value.Transformer
	reporter    metrics.StatsReporter
}

func NewKMSv2ServerWrapped(clusterSeed []byte) (*KeyManagementServiceV2ServerWrapped, error) {
	statsReporter, err := metrics.NewStatsReporter()
	if err != nil {
		return nil, fmt.Errorf("failed to create stats reporter: %w", err)
	}

	transformer, err := aes.NewHKDFExtendedNonceGCMTransformer(clusterSeed)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new HKDF extended nonce transformer: %w", err)
	}

	return &KeyManagementServiceV2ServerWrapped{
		transformer: transformer,
		reporter:    statsReporter,
	}, nil
}

func (s *KeyManagementServiceV2ServerWrapped) Status(_ context.Context, _ *kmsv2.StatusRequest) (*kmsv2.StatusResponse, error) {
	return &kmsv2.StatusResponse{
		Version: version.KMSv2APIVersion,
		Healthz: "ok", // we are always healthy once we are bootstrapped
		KeyId:   staticKeyID,
	}, nil
}

func (s *KeyManagementServiceV2ServerWrapped) Encrypt(ctx context.Context, request *kmsv2.EncryptRequest) (_ *kmsv2.EncryptResponse, err error) {
	mlog.Debug("encrypt request received", "uid", request.Uid)
	start := time.Now()

	defer func() {
		errors := ""
		status := metrics.SuccessStatusTypeValue
		if err != nil {
			status = metrics.ErrorStatusTypeValue
			errors = err.Error()
		}
		s.reporter.ReportRequest(ctx, metrics.EncryptOperationTypeValue, status, time.Since(start).Seconds(), errors)
		mlog.Debug("encrypt request complete", "uid", request.Uid)
	}()

	// TODO decide if we should make our own authenticated data nonce
	//  this value is required because of the assumptions NewHKDFExtendedNonceGCMTransformer makes
	dataCtx := value.DefaultContext(request.Uid)

	ciphertext, err := s.transformer.TransformToStorage(ctx, request.Plaintext, dataCtx)
	if err != nil {
		mlog.Error("failed to encrypt", err, "uid", request.Uid)
		return &kmsv2.EncryptResponse{}, err
	}

	// TODO decide if we want any of this data, probably not?
	//  dateAnnotationKey:           []byte(result.Header.Get(dateAnnotationValue)),
	//  requestIDAnnotationKey:      []byte(result.Header.Get(requestIDAnnotationValue)),
	//  keyvaultRegionAnnotationKey: []byte(result.Header.Get(keyvaultRegionAnnotationValue)),
	//  algorithmAnnotationKey:      []byte(encryptionAlgorithm),
	return &kmsv2.EncryptResponse{
		Ciphertext: ciphertext,
		KeyId:      staticKeyID,
		Annotations: map[string][]byte{
			authenticatedDataAnnotationKey: dataCtx,
			versionAnnotationKey:           []byte(wrappedEncryptionResponseVersion),
		},
	}, nil
}

func (s *KeyManagementServiceV2ServerWrapped) Decrypt(ctx context.Context, request *kmsv2.DecryptRequest) (_ *kmsv2.DecryptResponse, err error) {
	mlog.Debug("decrypt request received", "uid", request.Uid)
	start := time.Now()

	defer func() {
		errors := ""
		status := metrics.SuccessStatusTypeValue
		if err != nil {
			status = metrics.ErrorStatusTypeValue
			errors = err.Error()
		}
		s.reporter.ReportRequest(ctx, metrics.DecryptOperationTypeValue, status, time.Since(start).Seconds(), errors)
		mlog.Debug("decrypt request complete", "uid", request.Uid)
	}()

	if err := validateWrappedAnnotations(request.Annotations, request.KeyId); err != nil {
		return nil, fmt.Errorf("failed to validate annotations: %w", err)
	}

	// staleness doesn't exist as a concept here so we ignore it
	plaintext, _, err := s.transformer.TransformFromStorage(ctx, request.Ciphertext, value.DefaultContext(request.Annotations[authenticatedDataAnnotationKey]))
	if err != nil {
		mlog.Error("failed to decrypt", err, "uid", request.Uid)
		return &kmsv2.DecryptResponse{}, err
	}

	return &kmsv2.DecryptResponse{
		Plaintext: plaintext,
	}, nil
}

func validateWrappedAnnotations(annotations map[string][]byte, keyID string) error {
	if keyID != staticKeyID {
		return fmt.Errorf(
			"key id %s does not match expected key id %s used for encryption",
			keyID,
			staticKeyID,
		)
	}

	if len(annotations) == 0 {
		return fmt.Errorf("invalid annotations, annotations cannot be empty")
	}

	if len(annotations[authenticatedDataAnnotationKey]) == 0 {
		return fmt.Errorf("missing authenticated data annotation value")
	}

	if dataVersion := string(annotations[versionAnnotationKey]); dataVersion != wrappedEncryptionResponseVersion {
		return fmt.Errorf(
			"version %s does not match expected version %s used for encryption",
			dataVersion,
			wrappedEncryptionResponseVersion,
		)
	}

	return nil
}

func BlockingRunAlwaysHealthyServer(healthCheckURL *url.URL) {
	serveMux := http.NewServeMux()
	serveMux.HandleFunc(healthCheckURL.EscapedPath(), func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	server := &http.Server{
		Addr:              healthCheckURL.Host,
		ReadHeaderTimeout: 5 * time.Second,
		Handler:           serveMux,
	}
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		mlog.Fatal(err, "failed to start health check server", "url", healthCheckURL.String())
	}
}