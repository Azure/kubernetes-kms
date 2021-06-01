package utils

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/kubernetes-kms/pkg/metrics"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
)

// ParseEndpoint returns unix socket's protocol and address
func ParseEndpoint(ep string) (string, string, error) {
	if strings.HasPrefix(strings.ToLower(ep), "unix://") {
		s := strings.SplitN(ep, "://", 2)
		if s[1] != "" {
			return s[0], s[1], nil
		}
	}
	return "", "", fmt.Errorf("invalid endpoint: %v", ep)
}

// UnaryServerInterceptor provides metrics around Unary RPCs.
func UnaryServerInterceptor() func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		reporter := metrics.NewStatsReporter()

		var err error
		defer func() {
			if err != nil {
				reporter.ReportRequest(ctx, fmt.Sprintf("%s_%s", metrics.GrpcOperationTypeValue, getGRPCMethodName(info.FullMethod)), metrics.ErrorStatusTypeValue, time.Since(start).Seconds(), err.Error())
				return
			}
			reporter.ReportRequest(ctx, fmt.Sprintf("%s_%s", metrics.GrpcOperationTypeValue, getGRPCMethodName(info.FullMethod)), metrics.SuccessStatusTypeValue, time.Since(start).Seconds())
		}()

		klog.V(5).Infof("GRPC call: %s", info.FullMethod)
		resp, err := handler(ctx, req)
		if err != nil {
			klog.ErrorS(err, "GRPC request error")
		}
		return resp, err
	}
}

func getGRPCMethodName(fullMethodName string) string {
	fullMethodName = strings.TrimPrefix(fullMethodName, "/")
	methodNames := strings.Split(fullMethodName, "/")
	if len(methodNames) >= 2 {
		return strings.ToLower(methodNames[1])
	}

	return "unknown"
}
