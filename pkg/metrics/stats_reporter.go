package metrics

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
)

const (
	instrumentationName       = "keyvaultkms"
	errorTypeKey              = "error_type"
	statusTypeKey             = "status"
	errorStatusTypeValue      = "error"
	successStatusTypeValue    = "success"
	operationTypeKey          = "operation"
	encryptOperationTypeValue = "encrypt"
	decryptOperationTypeValue = "decrypt"
)

var (
	totalRequest    metric.Int64Counter
	requestDuration metric.Float64ValueRecorder
)

type reporter struct {
	meter metric.Meter
}

// StatsReporter reports metrics
type StatsReporter interface {
	ReportEncryptCountMetric(ctx context.Context)
	ReportDecryptCountMetric(ctx context.Context)
	ReportEncryptErrorCountMetric(ctx context.Context, errorMessage string)
	ReportDecryptErrorCountMetric(ctx context.Context, errorMessage string)
	ReportEncryptDurationMetric(ctx context.Context, duration float64)
	ReportDecryptDurationMetric(ctx context.Context, duration float64)
}

// NewStatsReporter instantiates otel reporter
func NewStatsReporter() StatsReporter {
	meter := global.Meter(instrumentationName)

	totalRequest = metric.Must(meter).NewInt64Counter("total_request", metric.WithDescription("Total number of requests"))
	requestDuration = metric.Must(meter).NewFloat64ValueRecorder("duration_seconds", metric.WithDescription("Distribution of how long it took for an operation"))

	return &reporter{
		meter: meter,
	}
}

func (r *reporter) ReportEncryptCountMetric(ctx context.Context) {
	labels := []attribute.KeyValue{
		attribute.String(operationTypeKey, encryptOperationTypeValue),
		attribute.String(statusTypeKey, successStatusTypeValue),
	}
	totalRequest.Add(ctx, 1, labels...)
}

func (r *reporter) ReportDecryptCountMetric(ctx context.Context) {
	labels := []attribute.KeyValue{
		attribute.String(operationTypeKey, decryptOperationTypeValue),
		attribute.String(statusTypeKey, successStatusTypeValue),
	}
	totalRequest.Add(ctx, 1, labels...)
}

func (r *reporter) ReportEncryptErrorCountMetric(ctx context.Context, errorMessage string) {
	labels := []attribute.KeyValue{
		attribute.String(errorTypeKey, errorMessage),
		attribute.String(operationTypeKey, encryptOperationTypeValue),
		attribute.String(statusTypeKey, errorStatusTypeValue),
	}
	totalRequest.Add(ctx, 1, labels...)
}

func (r *reporter) ReportDecryptErrorCountMetric(ctx context.Context, errorMessage string) {
	labels := []attribute.KeyValue{
		attribute.String(errorTypeKey, errorMessage),
		attribute.String(operationTypeKey, decryptOperationTypeValue),
		attribute.String(statusTypeKey, errorStatusTypeValue),
	}
	totalRequest.Add(ctx, 1, labels...)
}

func (r *reporter) ReportEncryptDurationMetric(ctx context.Context, duration float64) {
	r.meter.RecordBatch(ctx,
		[]attribute.KeyValue{
			attribute.String(operationTypeKey, encryptOperationTypeValue),
		},
		requestDuration.Measurement(duration),
	)
}

func (r *reporter) ReportDecryptDurationMetric(ctx context.Context, duration float64) {
	r.meter.RecordBatch(ctx,
		[]attribute.KeyValue{
			attribute.String(operationTypeKey, decryptOperationTypeValue),
		},
		requestDuration.Measurement(duration),
	)
}
