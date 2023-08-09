// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/Azure/kubernetes-kms/pkg/metrics"
	mockkeyvault "github.com/Azure/kubernetes-kms/pkg/plugin/mock_keyvault"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"google.golang.org/grpc"
	kmsv1 "k8s.io/kms/apis/v1beta1"
	kmsv2 "k8s.io/kms/apis/v2"
	"monis.app/mlog"
)

func TestServe(t *testing.T) {
	tests := []struct {
		desc                   string
		setEncryptResponse     string
		setDecryptResponse     string
		setEncryptError        error
		setDecryptError        error
		expectedHTTPStatusCode int
	}{
		{
			desc:                   "failed to encrypt in health check",
			setEncryptResponse:     "",
			setEncryptError:        fmt.Errorf("failed to encrypt"),
			expectedHTTPStatusCode: http.StatusServiceUnavailable,
		},
		{
			desc:                   "failed to decrypt in health check",
			setEncryptResponse:     "",
			setEncryptError:        nil,
			setDecryptResponse:     "",
			setDecryptError:        fmt.Errorf("failed to decrypt"),
			expectedHTTPStatusCode: http.StatusServiceUnavailable,
		},
		{
			desc:                   "encrypt-decrypt mismatch",
			setEncryptResponse:     "bar",
			setEncryptError:        nil,
			setDecryptResponse:     "foo",
			setDecryptError:        nil,
			expectedHTTPStatusCode: http.StatusServiceUnavailable,
		},
		{
			desc:                   "successful health check",
			setEncryptResponse:     "bar",
			setDecryptResponse:     "healthcheck",
			expectedHTTPStatusCode: http.StatusOK,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			socketPath := fmt.Sprintf("%s/kms.sock", getTempTestDir(t))
			defer os.Remove(socketPath)

			fakeKMSServer, fakeKMSV2Server, mockKVClient, err := setupFakeKMSServer(socketPath)
			if err != nil {
				t.Fatalf("failed to create fake kms server, err: %+v", err)
			}

			mockKVClient.SetEncryptResponse([]byte(test.setEncryptResponse), test.setEncryptError)
			mockKVClient.SetDecryptResponse([]byte(test.setDecryptResponse), test.setDecryptError)

			healthz := &HealthZ{
				KMSv1Server:    fakeKMSServer,
				KMSv2Server:    fakeKMSV2Server,
				UnixSocketPath: socketPath,
				RPCTimeout:     20 * time.Second,
				HealthCheckURL: &url.URL{
					Scheme: "http",
					Host:   net.JoinHostPort("localhost", "8080"),
					Path:   "/healthz",
				},
			}

			server := httptest.NewServer(healthz)
			defer server.Close()

			respCode, body := doHealthCheck(t, server.URL)
			if respCode != test.expectedHTTPStatusCode {
				t.Fatalf("expected status code: %v, got: %v", test.expectedHTTPStatusCode, respCode)
			}
			if test.expectedHTTPStatusCode == http.StatusOK && string(body) != "ok" {
				t.Fatalf("expected response body to be 'ok', got: %s", string(body))
			}
		})
	}
}

func TestCheckRPC(t *testing.T) {
	socketPath := fmt.Sprintf("%s/kms.sock", getTempTestDir(t))
	defer os.Remove(socketPath)

	fakeKMSV1Server, fakeKMSV2Server, mockKVClient, err := setupFakeKMSServer(socketPath)
	if err != nil {
		t.Fatalf("failed to create fake kms server, err: %+v", err)
	}
	healthz := &HealthZ{
		KMSv1Server:    fakeKMSV1Server,
		KMSv2Server:    fakeKMSV2Server,
		UnixSocketPath: socketPath,
	}
	mockKVClient.SetEncryptResponse([]byte(healthCheckPlainText), nil)
	mockKVClient.SetDecryptResponse([]byte(healthCheckPlainText), nil)

	conn, err := healthz.dialUnixSocket()
	if err != nil {
		t.Fatalf("failed to create connection, err: %+v", err)
	}

	err = healthz.checkRPC(
		context.TODO(),
		kmsv1.NewKeyManagementServiceClient(conn),
		kmsv2.NewKeyManagementServiceClient(conn),
	)
	if err != nil {
		t.Fatalf("expected err to be nil, got: %+v", err)
	}
}

func getTempTestDir(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "ut")
	if err != nil {
		t.Fatalf("expected err to be nil, got: %+v", err)
	}
	return tmpDir
}

func setupFakeKMSServer(socketPath string) (
	*KeyManagementServiceServer,
	*KeyManagementServiceV2Server,
	*mockkeyvault.KeyVaultClient,
	error,
) {
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, nil, nil, err
	}

	statsReporter, err := metrics.NewStatsReporter()
	if err != nil {
		return nil, nil, nil, err
	}

	kvClient := &mockkeyvault.KeyVaultClient{
		KeyID:     "mock-key-id",
		Algorithm: keyvault.RSA15,
	}
	fakeKMSV1Server := &KeyManagementServiceServer{
		kvClient: kvClient,
		reporter: statsReporter,
	}

	fakeKMSV2Server := &KeyManagementServiceV2Server{
		kvClient: kvClient,
		reporter: statsReporter,
	}

	s := grpc.NewServer()
	kmsv1.RegisterKeyManagementServiceServer(s, fakeKMSV1Server)
	kmsv2.RegisterKeyManagementServiceServer(s, fakeKMSV2Server)
	go func() {
		if err := s.Serve(listener); err != nil {
			mlog.Fatal(err, "failed to serve fake kms server")
		}
	}()

	return fakeKMSV1Server, fakeKMSV2Server, kvClient, nil
}

func doHealthCheck(t *testing.T, url string) (int, []byte) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("failed to create new http request, err: %+v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to invoke http request, err: %+v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body, err: %+v", err)
	}
	return resp.StatusCode, body
}
