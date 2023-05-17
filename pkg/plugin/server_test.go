// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/Azure/kubernetes-kms/pkg/metrics"
	mockkeyvault "github.com/Azure/kubernetes-kms/pkg/plugin/mock_keyvault"
	"github.com/Azure/kubernetes-kms/pkg/version"

	kmsv1 "k8s.io/kms/apis/v1beta1"
)

func TestEncrypt(t *testing.T) {
	tests := []struct {
		desc   string
		input  []byte
		output []byte
		err    error
	}{
		{
			desc:   "failed to encrypt",
			input:  []byte("foo"),
			output: []byte{},
			err:    fmt.Errorf("failed to encrypt"),
		},
		{
			desc:   "successfully encrypted",
			input:  []byte("foo"),
			output: []byte("bar"),
			err:    nil,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			kvClient := &mockkeyvault.KeyVaultClient{}
			kvClient.SetEncryptResponse(test.output, test.err)

			statsReporter, err := metrics.NewStatsReporter()
			if err != nil {
				t.Fatalf("failed to create stats reporter: %v", err)
			}

			kmsServer := KeyManagementServiceServer{
				kvClient: kvClient,
				reporter: statsReporter,
			}

			out, err := kmsServer.Encrypt(context.TODO(), &kmsv1.EncryptRequest{
				Plain: test.input,
			})
			if err != test.err {
				t.Fatalf("expected err: %v, got: %v", test.err, err)
			}
			if !bytes.Equal(out.GetCipher(), test.output) {
				t.Fatalf("expected out: %v, got: %v", test.output, out)
			}
		})
	}
}

func TestDecrypt(t *testing.T) {
	tests := []struct {
		desc   string
		input  []byte
		output []byte
		err    error
	}{
		{
			desc:   "failed to decrypt",
			input:  []byte("foo"),
			output: []byte{},
			err:    fmt.Errorf("failed to decrypt"),
		},
		{
			desc:   "successfully decrypted",
			input:  []byte("bar"),
			output: []byte("foo"),
			err:    nil,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			kvClient := &mockkeyvault.KeyVaultClient{}
			kvClient.SetDecryptResponse(test.output, test.err)

			statsReporter, err := metrics.NewStatsReporter()
			if err != nil {
				t.Fatalf("failed to create stats reporter: %v", err)
			}

			kmsServer := KeyManagementServiceServer{
				kvClient: kvClient,
				reporter: statsReporter,
			}

			out, err := kmsServer.Decrypt(context.TODO(), &kmsv1.DecryptRequest{
				Cipher: test.input,
			})
			if err != test.err {
				t.Fatalf("expected err: %v, got: %v", test.err, err)
			}
			if !bytes.Equal(out.GetPlain(), test.output) {
				t.Fatalf("expected out: %v, got: %v", test.output, out)
			}
		})
	}
}

func TestVersion(t *testing.T) {
	kmsServer := KeyManagementServiceServer{}

	version.BuildVersion = "latest"

	v, err := kmsServer.Version(context.TODO(), &kmsv1.VersionRequest{})
	if err != nil {
		t.Fatalf("expected err to be nil, got: %v", err)
	}
	if v.Version != version.KMSv1APIVersion {
		t.Fatalf("expected version: %s, got: %s", version.KMSv1APIVersion, v.Version)
	}
	if v.RuntimeName != version.Runtime {
		t.Fatalf("expected runtime: %s, got: %s", version.Runtime, v.RuntimeName)
	}
	if v.RuntimeVersion != "latest" {
		t.Fatalf("expected runtime version: %s, got: %s", version.BuildVersion, v.Version)
	}
}
