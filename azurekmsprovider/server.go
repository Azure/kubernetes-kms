package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"log"

	//"github.com/golang/glog"

	"golang.org/x/net/context"
	"golang.org/x/net/trace"
	// "golang.org/x/oauth2/google"

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

	cipher := base64.StdEncoding.EncodeToString(request.Plain)
	return &k8spb.EncryptResponse{Cipher: []byte(cipher)}, nil
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
