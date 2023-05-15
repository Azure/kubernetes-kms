package test

import (
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/apimachinery/pkg/util/uuid"
	kmsv1 "k8s.io/kms/apis/v1beta1"
	kmsv2 "k8s.io/kms/apis/v2"
)

const (
	netProtocol      = "unix"
	pathToUnixSocket = "/opt/azurekms.sock"
	version          = "v1beta1"
)

var (
	v1Client   kmsv1.KeyManagementServiceClient
	v2Client   kmsv2.KeyManagementServiceClient
	connection *grpc.ClientConn
	t          *testing.T
	err        error
)

func setupTestCase() {
	if t != nil {
		t.Log("setup test case")
		connection, err = newUnixSocketConnection(pathToUnixSocket)
		if err != nil {
			fmt.Printf("%s", err)
		}

		v1Client = kmsv1.NewKeyManagementServiceClient(connection)
		v2Client = kmsv2.NewKeyManagementServiceClient(connection)
	}
}

func teardownTestCase() {
	if t != nil {
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

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			t.Cleanup(cancel)

			v1EncryptRequest := kmsv1.EncryptRequest{Version: version, Plain: tc.want}
			v1EncryptResponse, err := v1Client.Encrypt(ctx, &v1EncryptRequest)
			if err != nil {
				t.Fatalf("encrypt request for KMS v1 failed with error: %+v", err)
			}

			v1DecryptRequest := kmsv1.DecryptRequest{Version: version, Cipher: v1EncryptResponse.Cipher}
			v1DecryptResponse, err := v1Client.Decrypt(ctx, &v1DecryptRequest)
			if !bytes.Equal(v1DecryptResponse.Plain, tc.want) {
				t.Fatalf("Expected secret, but got %s - %v", string(v1DecryptResponse.Plain), err)
			}

			uid := "integration-test-" + string(uuid.NewUUID())
			v2EncryptRequest := kmsv2.EncryptRequest{
				Plaintext: tc.want,
				Uid:       uid,
			}
			v2EncryptResponse, err := v2Client.Encrypt(ctx, &v2EncryptRequest)
			if err != nil {
				t.Fatalf("encrypt request for KMS v2 failed with error: %+v", err)
			}
			if v2EncryptResponse.KeyId == "" {
				t.Fatalf("Returned KeyId is empty")
			}

			if v2EncryptResponse.Annotations == nil {
				t.Fatalf("Returned Annotations is nil")
			}

			v2DecryptRequest := kmsv2.DecryptRequest{
				Ciphertext:  v2EncryptResponse.Ciphertext,
				KeyId:       v2EncryptResponse.KeyId,
				Uid:         uid,
				Annotations: v2EncryptResponse.Annotations,
			}
			v2DecryptResponse, err := v2Client.Decrypt(ctx, &v2DecryptRequest)
			if !bytes.Equal(v2DecryptResponse.Plaintext, tc.want) {
				t.Fatalf("Expected secret, but got %s - %v", string(v2DecryptResponse.Plaintext), err)
			}
		})
	}
}

// Check the KMS provider API version.
// Only matching version is supported now.
func TestV1Version(t *testing.T) {
	cases := []struct {
		name     string
		want     string
		expected string
	}{
		{"v1beta1", "v1beta1", "v1beta1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			t.Cleanup(cancel)

			request := &kmsv1.VersionRequest{Version: tc.want}
			response, err := v1Client.Version(ctx, request)
			if err != nil {
				t.Fatalf("failed get version from remote KMS provider: %v", err)
			}
			if response.Version != tc.want {
				t.Fatalf("KMS provider api version %s is not supported, only %s is supported now", tc.want, version)
			}
		})
	}
}

func TestV2Version(t *testing.T) {
	cases := []struct {
		name     string
		want     string
		expected string
	}{
		{"v2beta1", "v2beta1", "v2beta1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			t.Cleanup(cancel)

			request := &kmsv2.StatusRequest{}
			response, err := v2Client.Status(ctx, request)
			if err != nil {
				t.Fatalf("failed get status of remote KMS v2 provider: %v", err)
			}
			if response.Version != tc.want {
				t.Fatalf("KMS v2 provider api version %s is not supported, only %s is supported now", tc.want, version)
			}
		})
	}
}

func TestMain(m *testing.M) {
	t = &testing.T{}
	setupTestCase()
	m.Run()
	teardownTestCase()
}

func newUnixSocketConnection(path string) (*grpc.ClientConn, error) {
	addr := path
	dialer := func(ctx context.Context, addr string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, netProtocol, addr)
	}
	connection, err := grpc.Dial(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(dialer))
	if err != nil {
		return nil, err
	}
	return connection, nil
}
