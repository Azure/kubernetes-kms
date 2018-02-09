package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"log"

	//"github.com/golang/glog"

	"golang.org/x/net/context"
	"golang.org/x/net/trace"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	kv "github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest"

	k8spb "github.com/ritazh/k8s-azure-kms-plugin/v1beta1"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"os"
)

const (
	// Unix Domain Socket
	netProtocol    = "unix"
	version        = "v1beta1"
	runtime        = "Microsoft AzureKMS"
	runtimeVersion = "0.0.1"
)

type KMSServiceServer struct {
	pathToUnixSocket string
	net.Listener
	*grpc.Server
}

func getVaultsClient() keyvault.VaultsClient {
	token, _ := GetResourceManagementToken(AuthGrantType())
	vaultsClient := keyvault.NewVaultsClient(SubscriptionID())
	vaultsClient.Authorizer = autorest.NewBearerAuthorizer(token)
	return vaultsClient
}

func getKeysClient() kv.BaseClient {
	token, _ := GetKeyvaultToken(AuthGrantType())
	vmClient := kv.New()
	vmClient.Authorizer = token
	return vmClient
}

// GetVault returns an existing vault
func GetVault(ctx context.Context, vaultName string) (keyvault.Vault, error) {
	vaultsClient := getVaultsClient()
	return vaultsClient.Get(ctx, ResourceGroupName(), vaultName)
}

// GetKey returns an existing key
func doEncrypt(ctx context.Context, vaultName string, keyName string, data string) (*string, error) {
	vaultsClient := getKeysClient()
	// vault, err := GetVault(ctx, vaultName)
	// if err != nil {
	// 	fmt.Print("failed to get vault, error: %v", err)
	// 	return nil, err
	// }
	// fmt.Println(*vault.Name)

	sample := []byte("Hello, World!")
 	value := base64.RawURLEncoding.EncodeToString(sample)
	parameter := kv.KeyOperationsParameters {
		Algorithm: kv.RSA15,
		Value: &value,
	}
	
	result, err := vaultsClient.Encrypt(ctx, "https://ritaacikeyvault.vault.azure.net/", "testkey", "590a4737fe4a4eaeae3cd9df81f991fe", parameter)
	if err != nil {
		fmt.Print("failed to encrypt, error: %v", err)
		return nil, err
	}

	fmt.Println("result.Result")
	fmt.Println(result.Result)
	return result.Result, nil
}

func New(pathToUnixSocketFile string) *KMSServiceServer {
	// ctx := context.Background()

	kmsServiceServer := new(KMSServiceServer)
	kmsServiceServer.pathToUnixSocket = pathToUnixSocketFile
	fmt.Println(kmsServiceServer.pathToUnixSocket)
	
	return kmsServiceServer
}

func main() {

	var (
		debugListenAddr = flag.String("debug-listen-addr", "127.0.0.1:7901", "HTTP listen address.")
	)
	flag.Parse()
	err := parseArgs()
	if err != nil {
		log.Fatalf("failed to parse args: %s\n", err)
	}

	log.Println("KMSServiceServer service starting...")
	s := New("/tmp/test.socket")
	if err := s.cleanSockFile(); err != nil {
		fmt.Errorf("failed to clean sockfile, error: %v", err)
	}

	listener, err := net.Listen(netProtocol, s.pathToUnixSocket)
	if err != nil {
		fmt.Errorf("failed to start listener, error: %v", err)
	}
	s.Listener = listener

	server := grpc.NewServer()
	k8spb.RegisterKMSServiceServer(server, s)
	s.Server = server

	go server.Serve(listener)

	trace.AuthRequest = func(req *http.Request) (any, sensitive bool) { return true, true }
	log.Println("KMSServiceServer service started successfully.")
	log.Fatal(http.ListenAndServe(*debugListenAddr, nil))
}

func (s *KMSServiceServer) Version(ctx context.Context, request *k8spb.VersionRequest) (*k8spb.VersionResponse, error) {
	return &k8spb.VersionResponse{Version: version, RuntimeName: runtime, RuntimeVersion: runtimeVersion}, nil
}

func (s *KMSServiceServer) Encrypt(ctx context.Context, request *k8spb.EncryptRequest) (*k8spb.EncryptResponse, error) {

	fmt.Println("Processing EncryptRequest: ")
	///TODO
	cipher, err := doEncrypt(ctx, "ritaacikeyvault", "testkey", string(request.Plain))
	if err != nil {
		fmt.Print("failed to doencrypt, error: %v", err)
		return nil, err
	}
	// cipher := base64.StdEncoding.EncodeToString(request.Plain)
	return &k8spb.EncryptResponse{Cipher: []byte(*cipher)}, nil
}

func (s *KMSServiceServer) Decrypt(ctx context.Context, request *k8spb.DecryptRequest) (*k8spb.DecryptResponse, error) {

	fmt.Println("Processing DecryptRequest: ")

	///TODO

	plain, err := base64.StdEncoding.DecodeString(string(request.Cipher))
	if err != nil {
		fmt.Print("failed to decode, error: %v", err)
		return &k8spb.DecryptResponse{}, err
	}

	return &k8spb.DecryptResponse{Plain: []byte(plain)}, nil
}

func (s *KMSServiceServer) cleanSockFile() error {
	err := unix.Unlink(s.pathToUnixSocket)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete the socket file, error: %v", err)
	}
	return nil
}
