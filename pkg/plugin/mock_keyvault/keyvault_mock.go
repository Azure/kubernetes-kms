// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package mock_keyvault

import (
	"context"
	"sync"
)

type KeyVaultClient struct {
	mutex sync.Mutex

	encryptOut []byte
	encryptErr error
	decryptOut []byte
	decryptErr error
}

func (kvc *KeyVaultClient) Encrypt(ctx context.Context, cipher []byte) ([]byte, error) {
	kvc.mutex.Lock()
	defer kvc.mutex.Unlock()
	return kvc.encryptOut, kvc.encryptErr
}

func (kvc *KeyVaultClient) Decrypt(ctx context.Context, plain []byte) ([]byte, error) {
	kvc.mutex.Lock()
	defer kvc.mutex.Unlock()
	return kvc.decryptOut, kvc.decryptErr
}

func (kvc *KeyVaultClient) CheckIfKeyExists(ctx context.Context) error {
	return nil
}

func (kvc *KeyVaultClient) SetEncryptResponse(encryptOut []byte, err error) {
	kvc.mutex.Lock()
	defer kvc.mutex.Unlock()
	kvc.encryptOut = encryptOut
	kvc.encryptErr = err
	return
}

func (kvc *KeyVaultClient) SetDecryptResponse(decryptOut []byte, err error) {
	kvc.mutex.Lock()
	defer kvc.mutex.Unlock()
	kvc.decryptOut = decryptOut
	kvc.decryptErr = err
	return
}
