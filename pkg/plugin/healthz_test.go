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

	mockkeyvault "github.com/Azure/kubernetes-kms/pkg/plugin/mock_keyvault"

	"google.golang.org/grpc"
	pb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
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
			expectedHTTPStatusCode: http.StatusInternalServerError,
		},
		{
			desc:                   "failed to decrypt in health check",
			setEncryptResponse:     "",
			setEncryptError:        nil,
			setDecryptResponse:     "",
			setDecryptError:        fmt.Errorf("failed to decrypt"),
			expectedHTTPStatusCode: http.StatusInternalServerError,
		},
		{
			desc:                   "encrypt-decrypt mismatch",
			setEncryptResponse:     "bar",
			setEncryptError:        nil,
			setDecryptResponse:     "foo",
			setDecryptError:        nil,
			expectedHTTPStatusCode: http.StatusInternalServerError,
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

			fakeKMSServer, mockKVClient, err := setupFakeKMSServer(socketPath)
			if err != nil {
				t.Fatalf("failed to create fake kms server, err: %+v", err)
			}

			mockKVClient.SetEncryptResponse([]byte(test.setEncryptResponse), test.setEncryptError)
			mockKVClient.SetDecryptResponse([]byte(test.setDecryptResponse), test.setDecryptError)

			healthz := &HealthZ{
				KMSServer:      fakeKMSServer,
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

	fakeKMSServer, _, err := setupFakeKMSServer(socketPath)
	if err != nil {
		t.Fatalf("failed to create fake kms server, err: %+v", err)
	}
	healthz := &HealthZ{
		KMSServer:      fakeKMSServer,
		UnixSocketPath: socketPath,
	}

	conn, err := healthz.dialUnixSocket()
	if err != nil {
		t.Fatalf("failed to create connection, err: %+v", err)
	}
	err = healthz.checkRPC(context.TODO(), pb.NewKeyManagementServiceClient(conn))
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

func setupFakeKMSServer(socketPath string) (*KeyManagementServiceServer, *mockkeyvault.KeyVaultClient, error) {
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, nil, err
	}
	kvClient := &mockkeyvault.KeyVaultClient{}
	fakeKMSServer := &KeyManagementServiceServer{kvClient: kvClient}
	s := grpc.NewServer()
	pb.RegisterKeyManagementServiceServer(s, fakeKMSServer)
	go s.Serve(listener)

	return fakeKMSServer, kvClient, nil
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
