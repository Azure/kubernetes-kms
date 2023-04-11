// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package mockkeyvault

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

func (kvc *KeyVaultClient) Encrypt(_ context.Context, _ []byte) ([]byte, error) {
	kvc.mutex.Lock()
	defer kvc.mutex.Unlock()
	return kvc.encryptOut, kvc.encryptErr
}

func (kvc *KeyVaultClient) Decrypt(_ context.Context, _ []byte) ([]byte, error) {
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
