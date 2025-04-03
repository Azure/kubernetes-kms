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

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys"
	"github.com/Azure/kubernetes-kms/pkg/metrics"
	mockkeyvault "github.com/Azure/kubernetes-kms/pkg/plugin/mock_keyvault"

	"github.com/Azure/kubernetes-kms/pkg/version"
	kmsv2 "k8s.io/kms/apis/v2"
)

func TestV2Encrypt(t *testing.T) {
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
			kvClient := &mockkeyvault.KeyVaultClient{
				KeyID:     "mock-key-id",
				Algorithm: azkeys.EncryptionAlgorithmRSA15,
			}
			kvClient.SetEncryptResponse(test.output, test.err)

			statsReporter, err := metrics.NewStatsReporter()
			if err != nil {
				t.Fatalf("failed to create stats reporter: %v", err)
			}

			kmsV2Server := KeyManagementServiceV2Server{
				kvClient: kvClient,
				reporter: statsReporter,
			}

			out, err := kmsV2Server.Encrypt(context.TODO(), &kmsv2.EncryptRequest{
				Plaintext: test.input,
			})
			if err != test.err {
				t.Fatalf("expected err: %v, got: %v", test.err, err)
			}
			if !bytes.Equal(out.GetCiphertext(), test.output) {
				t.Fatalf("expected out: %v, got: %v", test.output, out)
			}
			if err == nil && (out.KeyId != kvClient.KeyID) {
				t.Fatalf("expected key id: %v, got: %v", kvClient.KeyID, out.KeyId)
			}
			if err == nil && (len(out.Annotations) == 0) {
				t.Fatalf("invalid annotations, annotations cannot be empty")
			}
		})
	}
}

func TestV2Decrypt(t *testing.T) {
	tests := []struct {
		desc        string
		input       []byte
		output      []byte
		err         error
		annotations map[string][]byte
	}{
		{
			desc:   "empty annotations failed to decrypt",
			input:  []byte("bar"),
			output: []byte{},
			err:    fmt.Errorf("invalid annotations, annotations cannot be empty"),
		},
		{
			desc:   "invalid keyid failed to decrypt",
			input:  []byte("bar"),
			output: []byte{},
			err:    fmt.Errorf("key id \"invalid-key-id\" does not match expected key id \"mock-key-id\" used for encryption"),
			annotations: map[string][]byte{
				algorithmAnnotationKey: []byte(azkeys.EncryptionAlgorithmRSA15),
				versionAnnotationKey:   []byte("1"),
			},
		},
		{
			desc:   "invalid algorithm failed to decrypt",
			input:  []byte("bar"),
			output: []byte{},
			err:    fmt.Errorf("algorithm \"insecure-algorithm\" does not match expected algorithm \"RSAOAEP256\" used for encryption"),
			annotations: map[string][]byte{
				algorithmAnnotationKey: []byte("insecure-algorithm"),
				versionAnnotationKey:   []byte("1"),
			},
		},
		{
			desc:   "invalid version failed to decrypt",
			input:  []byte("bar"),
			output: []byte{},
			err:    fmt.Errorf("version \"10\" does not match expected version \"1\" used for encryption"),
			annotations: map[string][]byte{
				algorithmAnnotationKey: []byte(azkeys.EncryptionAlgorithmRSA15),
				versionAnnotationKey:   []byte("10"),
			},
		},
		{
			desc:   "failed to decrypt",
			input:  []byte("foo"),
			output: []byte{},
			err:    fmt.Errorf("failed to decrypt"),
			annotations: map[string][]byte{
				algorithmAnnotationKey: []byte(azkeys.EncryptionAlgorithmRSA15),
				versionAnnotationKey:   []byte("1"),
			},
		},
		{
			desc:   "successfully decrypted",
			input:  []byte("bar"),
			output: []byte("foo"),
			err:    nil,
			annotations: map[string][]byte{
				algorithmAnnotationKey: []byte(azkeys.EncryptionAlgorithmRSA15),
				versionAnnotationKey:   []byte("1"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			kvClient := &mockkeyvault.KeyVaultClient{
				KeyID:     "mock-key-id",
				Algorithm: azkeys.EncryptionAlgorithmRSAOAEP256,
			}
			kvClient.SetDecryptResponse(test.output, test.err)

			statsReporter, err := metrics.NewStatsReporter()
			if err != nil {
				t.Fatalf("failed to create stats reporter: %v", err)
			}

			kmsV2Server := KeyManagementServiceV2Server{
				kvClient: kvClient,
				reporter: statsReporter,
			}

			out, err := kmsV2Server.Decrypt(context.TODO(), &kmsv2.DecryptRequest{
				Ciphertext:  test.input,
				Annotations: test.annotations,
				KeyId:       "mock-key-id",
			})
			if err != nil && (err.Error() != test.err.Error()) {
				t.Fatalf("expected err: %v, got: %v", test.err, err)
			}
			if !bytes.Equal(out.GetPlaintext(), test.output) {
				t.Fatalf("expected out: %v, got: %v", test.output, out)
			}
		})
	}
}

func TestStatus(t *testing.T) {
	kmsServer := KeyManagementServiceV2Server{}
	mockKeyVaultClient := &mockkeyvault.KeyVaultClient{
		KeyID: "mock-key-id",
	}
	mockKeyVaultClient.SetEncryptResponse([]byte(healthCheckPlainText), nil)
	mockKeyVaultClient.SetDecryptResponse([]byte(healthCheckPlainText), nil)
	kmsServer.kvClient = mockKeyVaultClient

	v, err := kmsServer.Status(context.TODO(), &kmsv2.StatusRequest{})
	if err != nil {
		t.Fatalf("expected err to be nil, got: %v", err)
	}

	if v.Version != version.KMSv2APIVersion {
		t.Fatalf("expected version: %s, got: %s", version.KMSv2APIVersion, v.Version)
	}

	if v.Healthz != "ok" {
		t.Fatalf("expected healthz response to be: %s, got: %s", "ok", v.Healthz)
	}

	if v.KeyId != "mock-key-id" {
		t.Fatalf("expected key id: %s, got: %s", "mock-key-id", v.KeyId)
	}
}
