package test

import (
	"fmt"
	"net"
	"testing"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	k8spb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
)

const (
	netProtocol      = "unix"
	pathToUnixSocket = "/opt/azurekms.socket"
	version          = "v1beta1"
)

var (
	client     k8spb.KeyManagementServiceClient
	connection *grpc.ClientConn
	err        error
)

func setupTestCase(t *testing.T) func(t *testing.T) {
	t.Log("setup test case")
	connection, err = newUnixSocketConnection(pathToUnixSocket)
	if err != nil {
		fmt.Printf("%s", err)
	}
	client = k8spb.NewKeyManagementServiceClient(connection)
	return func(t *testing.T) {
		t.Log("teardown test case")
		connection.Close()
	}
}

func TestEncryptDecrypt(t *testing.T) {
	cases := []struct {
		name     string
		want     []byte
		expected []byte
	}{
		{"text", []byte("secret"), []byte("secret")},
		{"number", []byte("1234"), []byte("1234")},
		{"special", []byte("!@#$%^&*()_"), []byte("!@#$%^&*()_")},
		{"GUID", []byte("b32a58c6-48c1-4552-8ff0-47680f3416d0"), []byte("b32a58c6-48c1-4552-8ff0-47680f3416d0")},
	}

	teardownTestCase := setupTestCase(t)
	defer teardownTestCase(t)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			encryptRequest := k8spb.EncryptRequest{Version: version, Plain: tc.want}
			encryptResponse, err := client.Encrypt(context.Background(), &encryptRequest)
			if err != nil {
				t.Fatalf("encrypt request failed with error: %+v", err)
			}

			decryptRequest := k8spb.DecryptRequest{Version: version, Cipher: encryptResponse.Cipher}
			decryptResponse, err := client.Decrypt(context.Background(), &decryptRequest)
			if string(decryptResponse.Plain) != string(tc.want) {
				t.Fatalf("Expected secret, but got %s - %v", string(decryptResponse.Plain), err)
			}
		})
	}
}

// Check the KMS provider API version.
// Only matching version is supported now.
func TestVersion(t *testing.T) {
	cases := []struct {
		name     string
		want     string
		expected string
	}{
		{"v1beta1", "v1beta1", "v1beta1"},
	}

	teardownTestCase := setupTestCase(t)
	defer teardownTestCase(t)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			request := &k8spb.VersionRequest{Version: tc.want}
			response, err := client.Version(context.Background(), request)
			if err != nil {
				t.Fatalf("failed get version from remote KMS provider: %v", err)
			}
			if response.Version != tc.want {
				t.Fatalf("KMS provider api version %s is not supported, only %s is supported now", tc.want, version)
			}
		})
	}
}

func newUnixSocketConnection(path string) (*grpc.ClientConn, error) {
	addr := path
	dialer := func(ctx context.Context, addr string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, netProtocol, addr)
	}
	connection, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithContextDialer(dialer))
	if err != nil {
		return nil, err
	}
	return connection, nil
}
