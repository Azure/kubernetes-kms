package main

import(
	// "log"
	// "testing"

	k8spb "github.com/ritazh/k8s-azure-kms/v1beta1"

	"golang.org/x/net/context"
	"fmt"
	"net"
	"time"
	"google.golang.org/grpc"
)

const (
	netProtocol    = "unix"
  	pathToUnixSocket = "/tmp/azurekms.socket"
  	timeout = 30 * time.Second
  	version = "v1beta1"
)

func main() {
	plainText := []byte("secret")

	connection, err := newUnixSocketConnection(pathToUnixSocket)
	if err != nil {
		fmt.Errorf("%v", err)
	}
	defer connection.Close()

	client := k8spb.NewKMSServiceClient(connection)

	err = checkAPIVersion(client)
 	if err != nil {
 		connection.Close()
 		fmt.Errorf("failed check version for %q, error: %v", pathToUnixSocket, err)
 	}

	encryptRequest := k8spb.EncryptRequest{Version: version, Plain: plainText}
	encryptResponse, err := client.Encrypt(context.Background(), &encryptRequest)
	
	if err != nil {
		fmt.Errorf("encrypt err: %v", err)
	}
	decryptRequest := k8spb.DecryptRequest{Version: version, Cipher: encryptResponse.Cipher}
	decryptResponse, err := client.Decrypt(context.Background(), &decryptRequest)
	if err != nil {
		fmt.Errorf("decrypt err: %v", err)
	}
	fmt.Println(string(decryptResponse.Plain))
	if string(decryptResponse.Plain) != string(plainText){
		fmt.Println("Expected secret, but got %s", string(decryptResponse.Plain))
	}
}

func newUnixSocketConnection(path string) (*grpc.ClientConn, error)  {
	addr := path
	dialer := func(addr string, timeout time.Duration) (net.Conn, error) {
		return net.DialTimeout(netProtocol, addr, timeout)
	}
	connection, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithDialer(dialer))
	if err != nil {
		return nil, err
	} else {
		fmt.Println("connection created")
	}

  return connection, nil
}

// Check the KMS provider API version.
// Only matching version is supported now.
func checkAPIVersion(kmsClient k8spb.KMSServiceClient) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	request := &k8spb.VersionRequest{Version: version}
	response, err := kmsClient.Version(ctx, request)
	if err != nil {
		return fmt.Errorf("failed get version from remote KMS provider: %v", err)
	}
	if response.Version != version {
		return fmt.Errorf("KMS provider api version %s is not supported, only %s is supported now",
			response.Version, version)
	}

	fmt.Println("KMS provider ", response.RuntimeName, "initialized, version:", response.RuntimeVersion)
	return nil
}
