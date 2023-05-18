package utils

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/kubernetes-kms/pkg/metrics"

	"google.golang.org/grpc"
	"monis.app/mlog"
)

// ParseEndpoint returns unix socket's protocol and address.
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
func UnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	var err error
	start := time.Now()
	reporter, err := metrics.NewStatsReporter()
	if err != nil {
		return nil, fmt.Errorf("failed to create stats reporter: %w", err)
	}

	defer func() {
		errors := ""
		status := metrics.SuccessStatusTypeValue
		if err != nil {
			status = metrics.ErrorStatusTypeValue
			errors = err.Error()
		}
		reporter.ReportRequest(ctx, fmt.Sprintf("%s_%s", metrics.GrpcOperationTypeValue, getGRPCMethodName(info.FullMethod)), status, time.Since(start).Seconds(), errors)
	}()

	mlog.Trace("GRPC call", "method", info.FullMethod)
	resp, err := handler(ctx, req)
	if err != nil {
		mlog.Error("GRPC request error", err)
	}
	return resp, err
}

func getGRPCMethodName(fullMethodName string) string {
	fullMethodName = strings.TrimPrefix(fullMethodName, "/")
	methodNames := strings.Split(fullMethodName, "/")
	if len(methodNames) >= 2 {
		return strings.ToLower(methodNames[1])
	}

	return "unknown"
}
