// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package mockkeyvault

import (
	"context"
	"fmt"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys"
	"k8s.io/kms/pkg/service"
)

type KeyVaultClient struct {
	mutex sync.Mutex

	encryptOut []byte
	encryptErr error
	decryptOut []byte
	decryptErr error
	KeyID      string
	Algorithm  azkeys.EncryptionAlgorithm
}

func (kvc *KeyVaultClient) Encrypt(_ context.Context, _ []byte, _ azkeys.EncryptionAlgorithm) (*service.EncryptResponse, error) {
	kvc.mutex.Lock()
	defer kvc.mutex.Unlock()
	return &service.EncryptResponse{
		Ciphertext: kvc.encryptOut,
		KeyID:      kvc.KeyID,
		Annotations: map[string][]byte{
			"key-id.azure.akv.io":    []byte(kvc.KeyID),
			"algorithm.azure.akv.io": []byte(kvc.Algorithm),
			"version.azure.akv.io":   []byte("1"),
		},
	}, kvc.encryptErr
}

func (kvc *KeyVaultClient) Decrypt(_ context.Context, _ []byte, _ azkeys.EncryptionAlgorithm, _ string, _ map[string][]byte, _ string) ([]byte, error) {
	kvc.mutex.Lock()
	defer kvc.mutex.Unlock()
	return kvc.decryptOut, kvc.decryptErr
}

func (kvc *KeyVaultClient) SetEncryptResponse(encryptOut []byte, err error) {
	kvc.mutex.Lock()
	defer kvc.mutex.Unlock()
	kvc.encryptOut = encryptOut
	kvc.encryptErr = err
}

func (kvc *KeyVaultClient) SetDecryptResponse(decryptOut []byte, err error) {
	kvc.mutex.Lock()
	defer kvc.mutex.Unlock()
	kvc.decryptOut = decryptOut
	kvc.decryptErr = err
}

func (kvc *KeyVaultClient) ValidateAnnotations(annotations map[string][]byte, keyID string) error {
	if len(annotations) == 0 {
		return fmt.Errorf("invalid annotations, annotations cannot be empty")
	}

	// validate key id
	if keyID != kvc.KeyID {
		return fmt.Errorf(
			"key id %q does not match expected key id %q used for encryption",
			string(annotations["key-id.azure.akv.io"]),
			kvc.KeyID,
		)
	}

	// validate algorithm
	if string(annotations["algorithm.azure.akv.io"]) != string(kvc.Algorithm) {
		return fmt.Errorf("algorithm %q does not match expected algorithm %q used for encryption", string(annotations["algorithm.azure.akv.io"]), kvc.Algorithm)
	}

	// validate version
	if string(annotations["version.azure.akv.io"]) != "1" {
		return fmt.Errorf(
			"version %q does not match expected version %q used for encryption",
			string(annotations["version.azure.akv.io"]),
			"1",
		)
	}

	return nil
}
