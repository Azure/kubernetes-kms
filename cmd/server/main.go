// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package main

import (
	"context"
	"flag"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Azure/kubernetes-kms/pkg/metrics"
	"github.com/Azure/kubernetes-kms/pkg/plugin"
	"github.com/Azure/kubernetes-kms/pkg/utils"
	"github.com/Azure/kubernetes-kms/pkg/version"

	"google.golang.org/grpc"
	pb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
	json "k8s.io/component-base/logs/json"
	"k8s.io/klog/v2"
)

var (
	listenAddr    = flag.String("listen-addr", "unix:///opt/azurekms.socket", "gRPC listen address")
	keyvaultName  = flag.String("keyvault-name", "", "Azure Key Vault name")
	keyName       = flag.String("key-name", "", "Azure Key Vault KMS key name")
	keyVersion    = flag.String("key-version", "", "Azure Key Vault KMS key version")
	logFormatJSON = flag.Bool("log-format-json", false, "set log formatter to json")
	// TODO remove this flag in future release.
	_              = flag.String("configFilePath", "/etc/kubernetes/azure.json", "[DEPRECATED] Path for Azure Cloud Provider config file")
	configFilePath = flag.String("config-file-path", "/etc/kubernetes/azure.json", "Path for Azure Cloud Provider config file")
	versionInfo    = flag.Bool("version", false, "Prints the version information")

	healthzPort    = flag.Int("healthz-port", 8787, "port for health check")
	healthzPath    = flag.String("healthz-path", "/healthz", "path for health check")
	healthzTimeout = flag.Duration("healthz-timeout", 20*time.Second, "RPC timeout for health check")
	metricsBackend = flag.String("metrics-backend", "prometheus", "Backend used for metrics")
	metricsAddress = flag.String("metrics-addr", "8095", "The address the metric endpoint binds to")

	proxyMode    = flag.Bool("proxy-mode", false, "Proxy mode")
	proxyAddress = flag.String("proxy-address", "", "proxy address")
	proxyPort    = flag.Int("proxy-port", 7788, "port for proxy")
)

func main() {
	klog.InitFlags(nil)

	flag.Parse()

	if *logFormatJSON {
		klog.SetLogger(json.JSONLogger)
	}

	if *versionInfo {
		version.PrintVersion()
		os.Exit(0)
	}

	ctx := withShutdownSignal(context.Background())

	// initialize metrics exporter
	err := metrics.InitMetricsExporter(*metricsBackend, *metricsAddress)
	if err != nil {
		klog.Fatalf("failed to initialize metrics exporter, error: %+v", err)
	}

	klog.InfoS("Starting KeyManagementServiceServer service", "version", version.BuildVersion, "buildDate", version.BuildDate)
	kmsServer, err := plugin.New(ctx, *configFilePath, *keyvaultName, *keyName, *keyVersion, *proxyMode, *proxyAddress, *proxyPort)
	if err != nil {
		klog.Fatalf("failed to create server, error: %v", err)
	}

	// Initialize and run the GRPC server
	proto, addr, err := utils.ParseEndpoint(*listenAddr)
	if err != nil {
		klog.Fatalf("failed to parse endpoint, err: %+v", err)
	}
	if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
		klog.Fatalf("failed to remove %s, error: %s", addr, err.Error())
	}

	listener, err := net.Listen(proto, addr)
	if err != nil {
		klog.Fatalf("failed to listen: %v", err)
	}
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(utils.UnaryServerInterceptor),
	}

	s := grpc.NewServer(opts...)
	pb.RegisterKeyManagementServiceServer(s, kmsServer)

	klog.Infof("Listening for connections on address: %v", listener.Addr())
	go s.Serve(listener)

	healthz := &plugin.HealthZ{
		KMSServer: kmsServer,
		HealthCheckURL: &url.URL{
			Host: net.JoinHostPort("", strconv.FormatUint(uint64(*healthzPort), 10)),
			Path: *healthzPath,
		},
		UnixSocketPath: listener.Addr().String(),
		RPCTimeout:     *healthzTimeout,
	}
	go healthz.Serve()

	<-ctx.Done()
	// gracefully stop the grpc server
	klog.Infof("terminating the server")
	s.GracefulStop()

	klog.Flush()
	// using os.Exit skips running deferred functions
	os.Exit(0)
}

// withShutdownSignal returns a copy of the parent context that will close if
// the process receives termination signals.
func withShutdownSignal(ctx context.Context) context.Context {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT, os.Interrupt)

	nctx, cancel := context.WithCancel(ctx)

	go func() {
		<-signalChan
		klog.Info("received shutdown signal")
		cancel()
	}()
	return nctx
}
