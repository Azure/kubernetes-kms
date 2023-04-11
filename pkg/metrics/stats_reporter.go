package metrics

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
)

const (
	instrumentationName  = "keyvaultkms"
	errorMessageKey      = "error_message"
	statusTypeKey        = "status"
	operationTypeKey     = "operation"
	kmsRequestMetricName = "kms_request"
	// ErrorStatusTypeValue sets status tag to "error".
	ErrorStatusTypeValue = "error"
	// SuccessStatusTypeValue sets status tag to "success".
	SuccessStatusTypeValue = "success"
	// EncryptOperationTypeValue sets operation tag to "encrypt".
	EncryptOperationTypeValue = "encrypt"
	// DecryptOperationTypeValue sets operation tag to "decrypt".
	DecryptOperationTypeValue = "decrypt"
	// GrpcOperationTypeValue sets operation tag to "grpc".
	GrpcOperationTypeValue = "grpc"
)

type reporter struct {
	histogram metric.Float64Histogram
}

// StatsReporter reports metrics.
type StatsReporter interface {
	ReportRequest(ctx context.Context, operationType, status string, duration float64, errors ...string)
}

// NewStatsReporter instantiates otel reporter.
func NewStatsReporter() (StatsReporter, error) {
	meter := global.Meter(instrumentationName)

	metricCounter, err := meter.Float64Histogram(
		kmsRequestMetricName,
		metric.WithDescription("Distribution of how long it took for an operation"),
	)
	if err != nil {
		return nil, err
	}

	return &reporter{
		histogram: metricCounter,
	}, nil
}

func (r *reporter) ReportRequest(ctx context.Context, operationType, status string, duration float64, errors ...string) {
	labels := []attribute.KeyValue{
		attribute.String(operationTypeKey, operationType),
		attribute.String(statusTypeKey, status),
	}

	// Add errors
	if (status == ErrorStatusTypeValue) && len(errors) > 0 {
		for _, err := range errors {
			labels = append(labels, attribute.String(errorMessageKey, err))
		}
	}

	r.histogram.Record(ctx, duration, metric.WithAttributes(labels...))
}
