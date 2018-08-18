// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package main

import (
	"time"
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

	storagemgmt "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2017-10-01/storage"
	storage "github.com/Azure/azure-sdk-for-go/storage"
	kvmgmt "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	kv "github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"

	k8spb "github.com/Azure/kubernetes-kms/v1beta1"
)

const (
	// Unix Domain Socket
	netProtocol         = "unix"
	socketPath          = "/opt/azurekms.socket"
	version             = "v1beta1"
	runtime             = "Microsoft AzureKMS"
	runtimeVersion		= "0.0.4"
	maxRetryTimeout		= 60
	retryIncrement		= 5
	azurePublicCloud    = "AzurePublicCloud"
)

type AzureConfig struct {
    Id    string `json:"id"`
    Value string `json:"value" binding:"required"`
}

// KeyManagementServiceServer is a gRPC server.
type KeyManagementServiceServer struct {
	*grpc.Server
	azConfig *AzureAuthConfig
	pathToUnixSocket string
	providerVaultName *string
	providerKeyName *string
	providerKeyVersion *string
	net.Listener
	configFilePath string
	env *azure.Environment
	resourceGroup *string
}

// New creates an instance of the KMS Service Server.
func New(pathToUnixSocketFile string, configFilePath string) (*KeyManagementServiceServer, error) {
	keyManagementServiceServer := new(KeyManagementServiceServer)
	keyManagementServiceServer.pathToUnixSocket = pathToUnixSocketFile
	keyManagementServiceServer.configFilePath = configFilePath
	keyManagementServiceServer.azConfig, _ = GetAzureAuthConfig(keyManagementServiceServer.configFilePath)
	if keyManagementServiceServer.azConfig.SubscriptionID == "" {
		return nil, fmt.Errorf("Missing SubscriptionID in azure config")
	}
	vaultName, keyName, keyVersion, resourceGroup, err := GetKMSProvider(keyManagementServiceServer.configFilePath)
	if err != nil {
		return nil, err
	}
	keyManagementServiceServer.providerVaultName = vaultName
	keyManagementServiceServer.providerKeyName = keyName
	keyManagementServiceServer.providerKeyVersion = keyVersion
	keyManagementServiceServer.resourceGroup = resourceGroup

	keyManagementServiceServer.env, err = GetCloudEnv(configFilePath)
	if err != nil {
		return nil, err
	}
	fmt.Println(keyManagementServiceServer.pathToUnixSocket)
	return keyManagementServiceServer, nil
}

func getKey(subscriptionID string, providerVaultName string, providerKeyName string, providerKeyVersion string, resourceGroup string, configFilePath string, env *azure.Environment) (kv.ManagementClient, string, string, string, error)  {
	kvClient := kv.New()

	vaultUrl, err := getVault(subscriptionID, providerVaultName, resourceGroup, configFilePath, env)
	if err != nil {
		return kvClient, "", "", "", fmt.Errorf("failed to get vault, error: %v", err)
	}

	token, err := GetKeyvaultToken(AuthGrantType(), configFilePath)
	if err != nil {
		return kvClient, "", "", "", fmt.Errorf("failed to get token, error: %v", err)
	}
	
	kvClient.Authorizer = token

	fmt.Println("Verify key version from key vault ", providerKeyName, providerKeyVersion, *vaultUrl)

	var kid *string
	keyBundle, err := kvClient.GetKey(*vaultUrl, providerKeyName, providerKeyVersion)
	if err != nil {
		if providerKeyVersion != "" {
			return kvClient, "", "", "", fmt.Errorf("failed to verify the provided key version, error: %v", err)
		}
		// when we are not able to verify the latest key version for keyName, create key
		kid, err = createKey(kvClient, *vaultUrl, providerKeyName, providerVaultName, resourceGroup, subscriptionID, configFilePath, env)
		if err != nil {
			fmt.Println("Err returned from createKey: ", err.Error())

			if strings.Contains(err.Error(), "LeaseAlreadyPresent") {
				fmt.Println("createKey failed LeaseAlreadyPresent")
				
				t := 0
				for t < maxRetryTimeout {
					keybundle, err := kvClient.GetKey(*vaultUrl, providerKeyName, "")
					if err == nil {
						kid = keybundle.Key.Kid
						break;
					} else {
						t += retryIncrement
						time.Sleep(retryIncrement * time.Second)
						fmt.Printf("sleep %d secs, retry t: %d secs. ", retryIncrement, t)
					}
				}
				if t >= maxRetryTimeout {
					return kvClient, "", "", "", fmt.Errorf("failed to get key within the maxRetryTimeout: %d seconds", maxRetryTimeout)
				}
			} else {
				return kvClient, "", "", "", fmt.Errorf("failed to create key, error: %v", err)
			}
		}
	} else {
		// when we get latest key version from api, not from config file
		if providerKeyVersion == "" {
			kid = keyBundle.Key.Kid
		}
	}
	// when we get new key id, update key version in config file
	if kid != nil {
		version, err := getVersionFromKid(kid)
		if err != nil {
			return kvClient, "", "", "", err
		}
		fmt.Println("found key version: ", version)
		// save keyversion to azure.json
		err = UpdateKMSProvider(configFilePath, version)
		if err != nil {
			return kvClient, "", "", "", err
		}
		return kvClient, *vaultUrl, providerKeyName, version, nil
	}
	
	return kvClient, *vaultUrl, providerKeyName, providerKeyVersion, nil
}

func getVaultsClient(subscriptionID string, configFilePath string, env *azure.Environment) kvmgmt.VaultsClient {
	vaultsClient := kvmgmt.NewVaultsClientWithBaseURI(env.ResourceManagerEndpoint, subscriptionID)
	token, _ := GetManagementToken(AuthGrantType(), configFilePath)
	vaultsClient.Authorizer = token
	return vaultsClient
}

func getVault(subscriptionID string, vaultName string, resourceGroup string, configFilePath string, env *azure.Environment) (vaultUrl *string, err error) {
	vaultsClient := getVaultsClient(subscriptionID, configFilePath, env)
	vault, err := vaultsClient.Get(resourceGroup, vaultName)
	if err != nil {
		return nil, fmt.Errorf("failed to get vault, error: %v", err)
	}
	return vault.Properties.VaultURI, nil
}

func createKey(keyClient kv.ManagementClient, vaultUrl string, keyName string, providerVaultName string, resourceGroup string, subscriptionID string, configFilePath string, env *azure.Environment) (*string, error) {
	fmt.Println("Key not found. Creating a new key...")
	storageAccountsClient := storagemgmt.NewAccountsClientWithBaseURI(env.ResourceManagerEndpoint, subscriptionID)
	token, _ := GetManagementToken(AuthGrantType(), configFilePath)
	storageAccountsClient.Authorizer = token
	storageAcctName := providerVaultName
	res, err := storageAccountsClient.ListKeys(resourceGroup, storageAcctName)
	if err != nil {
		return nil, err
	}
	storageKey := *(((*res.Keys)[0]).Value)
	
	var storageCli storage.Client

	if env.Name == azurePublicCloud {
		storageCli, err = storage.NewBasicClient(storageAcctName, storageKey)
	} else {
		storageCli, err = storage.NewBasicClientOnSovereignCloud(storageAcctName, storageKey, *env)
	}
	if err != nil {
		return nil, err
	}
	blobCli := storageCli.GetBlobService()
	// Get container
	cnt := blobCli.GetContainerReference(keyName)
	ok, err := cnt.Exists()
	if err != nil {
		return nil, err
	}
	if !ok {
		fmt.Println("creating container: ", keyName)
		// Create container
		options := storage.CreateContainerOptions{
			Access: storage.ContainerAccessTypeContainer,
		}
		_, err := cnt.CreateIfNotExists(&options)
		if err != nil {
			return nil, err
		}
	}
	// Get blob
	b := cnt.GetBlobReference(keyName)
	ok, err = b.Exists()
	if !ok {
		fmt.Println("creating blob: ", keyName)
		// Create blob
		err = b.CreateBlockBlob(nil)
		if err != nil {
			return nil, err
		}
	}
	// Acquiring lease on blob, if blob already has a lease, return err
	_, err = b.AcquireLease(60, "", nil)
	if err != nil {
		return nil, err
	}
	fmt.Println("acquired lease")
	// Create KV key
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
	fmt.Println("Created a new kms key")
	return key.Key.Kid, nil
}

func getVersionFromKid (kid *string) (version string, err error) {
	if kid == nil {
		return "", fmt.Errorf("Key id is nil")
	}
	version = to.String(kid)
	index := strings.LastIndex(version, "/" )
	if (index > -1 && index < len(version)-1) {
		version = version[index+1:]
	}
	if version == "" {
		return "", fmt.Errorf("failed to parse version from: %v", kid)
	}
	return version, nil
}

// doEncrypt encrypts with an existing key
func doEncrypt(ctx context.Context, data []byte, subscriptionID string, providerVaultName string, providerKeyName string, providerKeyVersion string, resourceGroup string, configFilePath string, env *azure.Environment) (*string, error) {
	kvClient, vaultBaseUrl, keyName, keyVersion, err := getKey(subscriptionID, providerVaultName, providerKeyName, providerKeyVersion, resourceGroup, configFilePath, env)

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
func doDecrypt(ctx context.Context, data string, subscriptionID string, providerVaultName string, providerKeyName string, providerKeyVersion string, resourceGroup string, configFilePath string, env *azure.Environment) ([]byte, error) {
	kvClient, vaultBaseUrl, keyName, keyVersion, err := getKey(subscriptionID, providerVaultName, providerKeyName, providerKeyVersion, resourceGroup, configFilePath, env)
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
	configFilePath := flag.String("configFilePath", "/etc/kubernetes/azure.json", "Path for Azure Cloud Provider config file. ")
	flag.Parse()

	if configFilePath == nil {
		log.Fatalf("Failed to retrieve configFilePath")
	}

	log.Println("KeyManagementServiceServer service starting...")
	s, err := New(socketPath, *configFilePath)
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
	cipher, err := doEncrypt(ctx, request.Plain, s.azConfig.SubscriptionID, *(s.providerVaultName), *(s.providerKeyName), *(s.providerKeyVersion), *(s.resourceGroup), s.configFilePath, s.env)
	if err != nil {
		fmt.Print("failed to doencrypt, error: ", err)
		return &k8spb.EncryptResponse{}, err
	}
	return &k8spb.EncryptResponse{Cipher: []byte(*cipher)}, nil
}

func (s *KeyManagementServiceServer) Decrypt(ctx context.Context, request *k8spb.DecryptRequest) (*k8spb.DecryptResponse, error) {

	fmt.Println("Processing DecryptRequest: ")
	plain, err := doDecrypt(ctx, string(request.Cipher), s.azConfig.SubscriptionID, *(s.providerVaultName), *(s.providerKeyName), *(s.providerKeyVersion), *(s.resourceGroup), s.configFilePath, s.env)
	if err != nil {
		fmt.Println("failed to decrypt, error: ", err)
		return &k8spb.DecryptResponse{}, err
	}
	return &k8spb.DecryptResponse{Plain: plain}, nil
}

func (s *KeyManagementServiceServer) cleanSockFile() error {
	err := unix.Unlink(s.pathToUnixSocket)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete the socket file, error: %v", err)
	}
	return nil
}
