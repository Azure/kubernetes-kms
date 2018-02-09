package main

import(
	// "log"
	// "testing"

	k8spb "github.com/ritazh/k8s-azure-kms-plugin/v1beta1"

	"golang.org/x/net/context"
	"fmt"
	"net"
	"time"
	"google.golang.org/grpc"
)

const (
  pathToUnixSocket = "/tmp/test.socket"
)

func main() {
	plainText := []byte("secret")
	version := "v1beta1"

	connection, err := newUnixSocketConnection(pathToUnixSocket)
	if err != nil {
		fmt.Errorf("%v", err)
	}
	defer connection.Close()

	client := k8spb.NewKMSServiceClient(connection)

	encryptRequest := k8spb.EncryptRequest{Version: version, Plain: plainText}
	encryptResponse, err := client.Encrypt(context.Background(), &encryptRequest)
	
	if err != nil {
		fmt.Errorf("encrypt err: %v", err)
	}
	decryptRequest := k8spb.DecryptRequest{Version: version, Cipher: encryptResponse.Cipher}
	decryptResponse, err := client.Decrypt(context.Background(), &decryptRequest)
	if err != nil {
		fmt.Errorf("%v", err)
	}
	fmt.Println(string(decryptResponse.Plain))
	if string(decryptResponse.Plain) != string(plainText){
		fmt.Println("Expected secret, but got %s", string(decryptResponse.Plain))
	}
}

func newUnixSocketConnection(path string) (*grpc.ClientConn, error)  {
	protocol, addr := "unix", path
	dialer := func(addr string, timeout time.Duration) (net.Conn, error) {
		return net.DialTimeout(protocol, addr, timeout)
	}
	connection, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithDialer(dialer))
	if err != nil {
		return nil, err
	} else {
		fmt.Println("connection created")
	}

  return connection, nil
}
