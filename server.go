// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/net/context"
	"golang.org/x/net/trace"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	kvmgmt "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	kv "github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest/to"

	k8spb "github.com/Azure/kubernetes-kms/v1beta1"
)

const (
	// Unix Domain Socket
	netProtocol    = "unix"
	socketPath	   = "/tmp/azurekms.socket"
	version        = "v1beta1"
	runtime        = "Microsoft AzureKMS"
	runtimeVersion = "0.0.1"
	configFilePath  = "/etc/kubernetes/azure.json"
	azureResourceUrl = "https://management.azure.com/"
	vaultResourceUrl = "https://vault.azure.net"
)

// KeyManagementServiceServer is a gRPC server.
type KeyManagementServiceServer struct {
	*grpc.Server
	azConfig *AzureAuthConfig
	pathToUnixSocket string
    providerVaultName *string
    providerKeyName *string
    providerKeyVersion *string
	net.Listener
}

// New creates an instance of the KMS Service Server.
func New(pathToUnixSocketFile string) (*KeyManagementServiceServer, error) {
	keyManagementServiceServer := new(KeyManagementServiceServer)
	keyManagementServiceServer.pathToUnixSocket = pathToUnixSocketFile
	fmt.Println(keyManagementServiceServer.pathToUnixSocket)
	keyManagementServiceServer.azConfig, _ = GetAzureAuthConfig(configFilePath)
	if keyManagementServiceServer.azConfig.SubscriptionID == "" {
		return nil, fmt.Errorf("Missing SubscriptionID in azure config")
	}
	vaultName, keyName, keyVersion, err := GetKMSProvider(configFilePath)
	if err != nil {
		return nil, err
	}
	keyManagementServiceServer.providerVaultName = vaultName
	keyManagementServiceServer.providerKeyName = keyName
	keyManagementServiceServer.providerKeyVersion = keyVersion

	return keyManagementServiceServer, nil
}

func getKey(subscriptionID string, providerVaultName string, providerKeyName string, providerKeyVersion string) (kv.ManagementClient, string, string, string, error)  {
	kvClient := kv.New()

	vaultUrl, err := getVault(subscriptionID, providerVaultName)
	if err != nil {
		return kvClient, "", "", "", fmt.Errorf("failed to get vault, error: %v", err)
	}
	
	token, err := GetKeyvaultToken(AuthGrantType(), configFilePath, vaultResourceUrl)
	if err != nil {
		return kvClient, "", "", "", fmt.Errorf("failed to get token, error: %v", err)
	}
	
	kvClient.Authorizer = token

	fmt.Println("Verify key version from key vault ", providerKeyName, providerKeyVersion, *vaultUrl)
	_, err = kvClient.GetKey(*vaultUrl, providerKeyName, providerKeyVersion)
	if err != nil {
		if providerKeyVersion != "" {
			return kvClient, "", "", "", fmt.Errorf("failed to verify the provided key version, error: %v", err)
		}
		// when we are not able to verify the latest key version for keyName, create key
		keyVersion, err := createKey(kvClient, *vaultUrl, providerKeyName)
		if err != nil {
			return kvClient, "", "", "", fmt.Errorf("failed to create key, error: %v", err)
		}
		version := to.String(keyVersion)
		index := strings.LastIndex(version, "/" )
		if (index > -1 && index < len(version)-1) {
			version = version[index+1:]
			fmt.Println(version)
		}
		return kvClient, *vaultUrl, providerKeyName, version, nil

	}
	
	return kvClient, *vaultUrl, providerKeyName, providerKeyVersion, nil
}

func getVaultsClient(subscriptionID string) kvmgmt.VaultsClient {
	
	vaultsClient := kvmgmt.NewVaultsClient(subscriptionID)
	token, _ := GetKeyvaultToken(AuthGrantType(), configFilePath, azureResourceUrl)
	vaultsClient.Authorizer = token
	return vaultsClient
}

func getVault(subscriptionID string, vaultName string) (vaultUrl *string, err error) {
	vaultsClient := getVaultsClient(subscriptionID)
	resourceGroup, err := GetResourceGroup(configFilePath)
	vault, err := vaultsClient.Get(*resourceGroup, vaultName)
	if err != nil {
		return nil, fmt.Errorf("failed to get vault, error: %v", err)
	}
	fmt.Println(to.String(vault.Properties.VaultURI))
	return vault.Properties.VaultURI, nil
}

func createKey(keyClient kv.ManagementClient, vaultUrl string, keyName string) (*string, error) {
	fmt.Println("Key not found. Creating a new key...")
	key, err := keyClient.CreateKey(
		vaultUrl,
		keyName,
		kv.KeyCreateParameters{
			KeyAttributes: &kv.KeyAttributes{
				Enabled: to.BoolPtr(true),
			},
			KeySize: to.Int32Ptr(2048), 
			KeyOps: &[]kv.JSONWebKeyOperation{
				kv.Encrypt,
				kv.Decrypt,
			},
			Kty: kv.RSA,
		})
	if err != nil {
		return nil, err
	}
	fmt.Println("Created new kms key")
	return key.Key.Kid, nil
}

// doEncrypt encrypts with an existing key
func doEncrypt(ctx context.Context, data []byte, subscriptionID string, providerVaultName string, providerKeyName string, providerKeyVersion string) (*string, error) {
	kvClient, vaultBaseUrl, keyName, keyVersion, err := getKey(subscriptionID, providerVaultName, providerKeyName, providerKeyVersion)

	if err != nil {
		return nil, err
	}

 	value := base64.RawURLEncoding.EncodeToString(data)
	parameter := kv.KeyOperationsParameters {
		Algorithm: kv.RSA15,
		Value: &value,
	}
	
	result, err := kvClient.Encrypt(vaultBaseUrl, keyName, keyVersion, parameter)
	if err != nil {
		fmt.Println("Failed to encrypt, error: ", err)
		return nil, err
	}
	return result.Result, nil
}

// doDecrypt decrypts with an existing key
func doDecrypt(ctx context.Context, data string, subscriptionID string, providerVaultName string, providerKeyName string, providerKeyVersion string) ([]byte, error) {
	kvClient, vaultBaseUrl, keyName, keyVersion, err := getKey(subscriptionID, providerVaultName, providerKeyName, providerKeyVersion)
	if err != nil {
		return nil, err
	}
	parameter := kv.KeyOperationsParameters {
		Algorithm: kv.RSA15,
		Value: &data,
	}
	
	result, err := kvClient.Decrypt(vaultBaseUrl, keyName, keyVersion, parameter)
	if err != nil {
		fmt.Print("failed to decrypt, error: ", err)
		return nil, err
	}
	bytes, err := base64.RawURLEncoding.DecodeString(*result.Result)
	return bytes, nil
}

func main() {
	sigChan := make(chan os.Signal, 1)
	// register for SIGTERM (docker)
	signal.Notify(sigChan, syscall.SIGTERM)

	var (
		debugListenAddr = flag.String("debug-listen-addr", "127.0.0.1:7901", "HTTP listen address.")
	)
	flag.Parse()

	log.Println("KeyManagementServiceServer service starting...")
	s, err := New(socketPath)
	if err != nil {
		log.Fatalf("Failed to start, error: %v", err)
	}
	if err := s.cleanSockFile(); err != nil {
		log.Fatalf("Failed to clean sockfile, error: %v", err)
	}

	listener, err := net.Listen(netProtocol, s.pathToUnixSocket)
	if err != nil {
		log.Fatalf("Failed to start listener, error: %v", err)
	}
	s.Listener = listener

	server := grpc.NewServer()
	k8spb.RegisterKeyManagementServiceServer(server, s)
	s.Server = server

	go server.Serve(listener)
	trace.AuthRequest = func(req *http.Request) (any, sensitive bool) { return true, true }
	log.Println("KeyManagementServiceServer service started successfully.")

	go func() {
		for {
			s := <-sigChan
			if s == syscall.SIGTERM {
				fmt.Println("force stop")
				fmt.Println("Shutting down gRPC service...")
				server.GracefulStop()
				os.Exit(0)
			} 
		}
	}()

	log.Fatal(http.ListenAndServe(*debugListenAddr, nil))
}

func (s *KeyManagementServiceServer) Version(ctx context.Context, request *k8spb.VersionRequest) (*k8spb.VersionResponse, error) {
	fmt.Println(version)
	return &k8spb.VersionResponse{Version: version, RuntimeName: runtime, RuntimeVersion: runtimeVersion}, nil
}

func (s *KeyManagementServiceServer) Encrypt(ctx context.Context, request *k8spb.EncryptRequest) (*k8spb.EncryptResponse, error) {

	fmt.Println("Processing EncryptRequest: ")
	cipher, err := doEncrypt(ctx, request.Plain, s.azConfig.SubscriptionID, *(s.providerVaultName), *(s.providerKeyName), *(s.providerKeyVersion))
	if err != nil {
		fmt.Print("failed to doencrypt, error: ", err)
		return &k8spb.EncryptResponse{}, err
	}
	return &k8spb.EncryptResponse{Cipher: []byte(*cipher)}, nil
}

func (s *KeyManagementServiceServer) Decrypt(ctx context.Context, request *k8spb.DecryptRequest) (*k8spb.DecryptResponse, error) {

	fmt.Println("Processing DecryptRequest: ")
	plain, err := doDecrypt(ctx, string(request.Cipher), s.azConfig.SubscriptionID, *(s.providerVaultName), *(s.providerKeyName), *(s.providerKeyVersion))
	if err != nil {
		fmt.Println("failed to decrypt, error: ", err)
		return &k8spb.DecryptResponse{}, err
	}
	return &k8spb.DecryptResponse{Plain: plain}, nil
}

func (s *KeyManagementServiceServer) cleanSockFile() error {
	err := unix.Unlink(s.pathToUnixSocket)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete the socket file, error: ", err)
	}
	return nil
}
