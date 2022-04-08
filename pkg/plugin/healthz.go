// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/Azure/kubernetes-kms/pkg/version"

	"google.golang.org/grpc"
	pb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
	"k8s.io/klog/v2"
)

const (
	healthCheckPlainText = "healthcheck"
)

type HealthZ struct {
	KMSServer      *KeyManagementServiceServer
	HealthCheckURL *url.URL
	UnixSocketPath string
	RPCTimeout     time.Duration
}

// Serve creates the http handler for serving health requests
func (h *HealthZ) Serve() {
	serveMux := http.NewServeMux()
	serveMux.HandleFunc(h.HealthCheckURL.EscapedPath(), h.ServeHTTP)
	if err := http.ListenAndServe(h.HealthCheckURL.Host, serveMux); err != nil && err != http.ErrServerClosed {
		klog.ErrorS(err, "failed to start health check server", "url", h.HealthCheckURL.String())
		os.Exit(1)
	}
}

func (h *HealthZ) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	klog.V(5).Info("Started health check")
	ctx, cancel := context.WithTimeout(context.Background(), h.RPCTimeout)
	defer cancel()

	conn, err := h.dialUnixSocket()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer conn.Close()

	// create the kms client
	kmsClient := pb.NewKeyManagementServiceClient(conn)
	// check version response against KMS-Plugin's gRPC endpoint.
	err = h.checkRPC(ctx, kmsClient)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	// check the configured keyvault, key, key version and permissions are still
	// valid to encrypt and decrypt with test data.
	enc, err := h.KMSServer.Encrypt(ctx, &pb.EncryptRequest{Plain: []byte(healthCheckPlainText)})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	dec, err := h.KMSServer.Decrypt(ctx, &pb.DecryptRequest{Cipher: enc.Cipher})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if string(dec.Plain) != healthCheckPlainText {
		http.Error(w, "plain text mismatch after decryption", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
	klog.V(5).Info("Completed health check")
}

// checkRPC initiates a grpc request to validate the socket is responding
// sends a KMS VersionRequest and checks if the VersionResponse is valid.
func (h *HealthZ) checkRPC(ctx context.Context, client pb.KeyManagementServiceClient) error {
	v, err := client.Version(ctx, &pb.VersionRequest{})
	if err != nil {
		return err
	}
	if v.Version != version.APIVersion || v.RuntimeName != version.Runtime || v.RuntimeVersion != version.BuildVersion {
		return fmt.Errorf("failed to get correct version response")
	}
	return nil
}

func (h *HealthZ) dialUnixSocket() (*grpc.ClientConn, error) {
	return grpc.Dial(
		h.UnixSocketPath,
		grpc.WithInsecure(),
		grpc.WithContextDialer(func(ctx context.Context, target string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", target)
		}),
	)
}
