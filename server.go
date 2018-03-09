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
	// "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	kv "github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	// "github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"

	k8spb "github.com/ritazh/k8s-azure-kms/v1beta1"
)

const (
	// Unix Domain Socket
	netProtocol    = "unix"
	socketPath	   = "/tmp/azurekms.socket"
	version        = "v1beta1"
	runtime        = "Microsoft AzureKMS"
	runtimeVersion = "0.0.1"
	configFilePath  = "/etc/kubernetes/azure.json"
	defaultKeyName = "k8sKMS"
	defaultVaultURI = "https://ritaacikeyvault.vault.azure.net/"
)

type KMSServiceServer struct {
	pathToUnixSocket string
	net.Listener
	*grpc.Server
}

func New(pathToUnixSocketFile string) *KMSServiceServer {
	kmsServiceServer := new(KMSServiceServer)
	kmsServiceServer.pathToUnixSocket = pathToUnixSocketFile
	fmt.Println(kmsServiceServer.pathToUnixSocket)
	return kmsServiceServer
}

func getKeysClient() (kv.ManagementClient, string, string, string, error)  {
	vmClient := kv.New()
	token, err := GetKeyvaultToken(AuthGrantType(), configFilePath)
	if err != nil {
		return vmClient, "", "", "", fmt.Errorf("failed to get token, error: %v", err)
	}
	
	vmClient.Authorizer = token
	vaultBaseURL, keyName, keyVersion, err := GetKMSProvider(configFilePath)
	if err != nil {
		fmt.Println("Unable to find KMS provider in configs")
		keyVersion, err := getDefaultKey(vmClient, defaultVaultURI, defaultKeyName)
		if err != nil {
			return vmClient, "", "", "", fmt.Errorf("failed to get/create default key, error: %v", err)
		}
		version := to.String(keyVersion)
		index := strings.LastIndex(version, "/" )
		if (index > -1 && index < len(version)-1) {
			version = version[index+1:]
			fmt.Println(version)
		}
		return vmClient, defaultVaultURI, defaultKeyName, version, nil
	}
	return vmClient, *vaultBaseURL, *keyName, *keyVersion, nil
}

func getDefaultKey(keyClient kv.ManagementClient, vaultURL string, keyName string) (*string, error) {
	result, err := keyClient.GetKeyVersions(vaultURL, keyName, to.Int32Ptr(1))
	if err != nil {
		return nil, err
	}
	if result.Value != nil && len(* result.Value) > 0 {
		fmt.Println("Found existing kms key")
		version := (* result.Value)[0].Kid
		return version, nil
	}
	key, err := keyClient.CreateKey(
		vaultURL,
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
func doEncrypt(ctx context.Context, data []byte) (*string, error) {
	vaultsClient, vaultBaseURL, keyName, keyVersion, err := getKeysClient()

	if err != nil {
		return nil, err
	}

 	value := base64.RawURLEncoding.EncodeToString(data)
	parameter := kv.KeyOperationsParameters {
		Algorithm: kv.RSA15,
		Value: &value,
	}
	
	result, err := vaultsClient.Encrypt(vaultBaseURL, keyName, keyVersion, parameter)
	if err != nil {
		fmt.Println("Failed to encrypt, error: ", err)
		return nil, err
	}
	return result.Result, nil
}

// doDecrypt decrypts with an existing key
func doDecrypt(ctx context.Context, data string) ([]byte, error) {
	vaultsClient, vaultBaseURL, keyName, keyVersion, err := getKeysClient()
	if err != nil {
		return nil, err
	}
	parameter := kv.KeyOperationsParameters {
		Algorithm: kv.RSA15,
		Value: &data,
	}
	
	result, err := vaultsClient.Decrypt(vaultBaseURL, keyName, keyVersion, parameter)
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

	log.Println("KMSServiceServer service starting...")
	s := New(socketPath)
	if err := s.cleanSockFile(); err != nil {
		fmt.Errorf("Failed to clean sockfile, error: ", err)
	}

	listener, err := net.Listen(netProtocol, s.pathToUnixSocket)
	if err != nil {
		fmt.Errorf("Failed to start listener, error: ", err)
	}
	s.Listener = listener

	server := grpc.NewServer()
	k8spb.RegisterKMSServiceServer(server, s)
	s.Server = server

	go server.Serve(listener)
	trace.AuthRequest = func(req *http.Request) (any, sensitive bool) { return true, true }
	log.Println("KMSServiceServer service started successfully.")

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

func (s *KMSServiceServer) Version(ctx context.Context, request *k8spb.VersionRequest) (*k8spb.VersionResponse, error) {
	fmt.Println(version)
	return &k8spb.VersionResponse{Version: version, RuntimeName: runtime, RuntimeVersion: runtimeVersion}, nil
}

func (s *KMSServiceServer) Encrypt(ctx context.Context, request *k8spb.EncryptRequest) (*k8spb.EncryptResponse, error) {

	fmt.Println("Processing EncryptRequest: ")
	cipher, err := doEncrypt(ctx, request.Plain)
	if err != nil {
		fmt.Print("failed to doencrypt, error: ", err)
		return &k8spb.EncryptResponse{}, err
	}
	return &k8spb.EncryptResponse{Cipher: []byte(*cipher)}, nil
}

func (s *KMSServiceServer) Decrypt(ctx context.Context, request *k8spb.DecryptRequest) (*k8spb.DecryptResponse, error) {

	fmt.Println("Processing DecryptRequest: ")
	plain, err := doDecrypt(ctx, string(request.Cipher))
	if err != nil {
		fmt.Println("failed to decrypt, error: ", err)
		return &k8spb.DecryptResponse{}, err
	}
	return &k8spb.DecryptResponse{Plain: plain}, nil
}

func (s *KMSServiceServer) cleanSockFile() error {
	err := unix.Unlink(s.pathToUnixSocket)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete the socket file, error: ", err)
	}
	return nil
}
