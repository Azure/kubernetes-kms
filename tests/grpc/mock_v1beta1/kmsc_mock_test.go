package mock_v1beta1_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"fmt"
	"testing"

	kmscmock "github.com/Azure/kubernetes-kms/tests/grpc/mock_v1beta1"
	k8spb "github.com/Azure/kubernetes-kms/v1beta1"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
)

var (
	// TODO get the version automatically from the folder name
	version = "v1beta1"
)

// func setup(t *testing.T) (*gomock.Controller, kmscmock.MockKeyManagementServiceClient) {
// 	ctrl := gomock.NewController(t)
// 	return nil, kmscmock.NewMockKeyManagementServiceClient(&ctrl)

// }

// rpcMsg implements the gomock.Matcher interface
type rpcMsg struct {
	msg proto.Message
}

func (r *rpcMsg) Matches(msg interface{}) bool {
	m, ok := msg.(proto.Message)
	if !ok {
		return false
	}
	return proto.Equal(m, r.msg)
}

func (r *rpcMsg) String() string {
	return fmt.Sprintf("is %s", r.msg)
}

//Version
func TestVersion(t *testing.T) {
	//ctrl, mockKeyManagementServiceClient := setup(t)
	ctrl := gomock.NewController(t)
	mockKeyManagementServiceClient := kmscmock.NewMockKeyManagementServiceClient(ctrl)
	defer ctrl.Finish()
	req := &k8spb.VersionRequest{Version: version}
	mockKeyManagementServiceClient.EXPECT().Version(
		gomock.Any(),
		&rpcMsg{msg: req},
	).Return(&k8spb.VersionResponse{Version: version}, nil)
	ask := "v1beta1"
	r, _ := mockKeyManagementServiceClient.Version(context.Background(), &k8spb.VersionRequest{Version: ask})
	if r != nil {
		t.Logf("Test passed, ask : %s answer : %s", version, ask)
	}
}

func TestBadVersion(t *testing.T) {
	//ctrl, mockKeyManagementServiceClient := setup(t)
	ctrl := gomock.NewController(t)
	mockKeyManagementServiceClient := kmscmock.NewMockKeyManagementServiceClient(ctrl)
	defer ctrl.Finish()
	mockKeyManagementServiceClient.EXPECT().Version(
		gomock.Any(),
		gomock.Any(),
	).Return(&k8spb.VersionResponse{Version: version}, errors.New("Error, Not Supported Version"))
	ask := "v1beta2"
	_, err := mockKeyManagementServiceClient.Version(context.Background(), &k8spb.VersionRequest{Version: ask})
	if err != nil {
		t.Logf(err.Error())
	}
}

//Encrypt
func TestEncrypt(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	//generate a random cipher
	genPrivateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	genPublicKey := &genPrivateKey.PublicKey
	message := base64.RawURLEncoding.EncodeToString([]byte("secret"))
	ciphertext, _ := rsa.EncryptPKCS1v15(rand.Reader, genPublicKey, []byte(message))

	mockKeyManagementServiceClient := kmscmock.NewMockKeyManagementServiceClient(ctrl)
	mockKeyManagementServiceClient.EXPECT().Encrypt(
		gomock.Any(),
		gomock.Any(),
	).Return(&k8spb.EncryptResponse{Cipher: ciphertext}, nil)
	_, err := mockKeyManagementServiceClient.Encrypt(context.Background(), &k8spb.EncryptRequest{Version: "v1beta1", Plain: []byte("secret")})
	if err != nil {
		t.Errorf("mocking failed")
	}
	t.Logf("Test passed")
	//t.Logf("Reply : %s", r.String())
}

// Decrypt

// Version

// Test with SPN, MSI,

// KeyVault not found, Key not found

// RG not there

// Test key rotation, using the latest version per default
// Encrypt and Decrypt with diff version
