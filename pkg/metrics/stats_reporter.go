package metrics

import (
	"context"
	"runtime"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
)

var (
	instrumentationName = "keyvaultkms"
	osTypeKey           = "os_type"
	errorTypeKey        = "error_type"
	encryptTotal        metric.Int64Counter
	decryptTotal        metric.Int64Counter
	encryptErrorTotal   metric.Int64Counter
	decryptErrorTotal   metric.Int64Counter
	encryptDuration     metric.Float64ValueRecorder
	decryptDuration     metric.Float64ValueRecorder
	runtimeOS           = runtime.GOOS
)

type reporter struct {
	meter metric.Meter
}

//StatsReporter reports metrics
type StatsReporter interface {
	ReportEncryptCountMetric(ctx context.Context)
	ReportDecryptCountMetric(ctx context.Context)
	ReportEncryptErrorCountMetric(ctx context.Context, errorType string)
	ReportDecryptErrorCountMetric(ctx context.Context, errorType string)
	ReportEncryptDurationMetric(ctx context.Context, duration float64)
	ReportDecryptDurationMetric(ctx context.Context, duration float64)
}

//NewStatsReporter instantiates otel reporter
func NewStatsReporter() StatsReporter {
	meter := global.Meter(instrumentationName)

	encryptTotal = metric.Must(meter).NewInt64Counter("total_encrypt", metric.WithDescription("Total number of Encrypt requests"))
	decryptTotal = metric.Must(meter).NewInt64Counter("total_decrypt", metric.WithDescription("Total number of Decrypt requests"))
	encryptErrorTotal = metric.Must(meter).NewInt64Counter("total_encrypt_error", metric.WithDescription("Total number of encrypt requests with error"))
	decryptErrorTotal = metric.Must(meter).NewInt64Counter("total_decrypt_error", metric.WithDescription("Total number of decrypt requests with error"))
	encryptDuration = metric.Must(meter).NewFloat64ValueRecorder("encrypt_duration_sec", metric.WithDescription("Distribution of how long it took to encrypt request"))
	decryptDuration = metric.Must(meter).NewFloat64ValueRecorder("decrypt_duration_sec", metric.WithDescription("Distribution of how long it took to decrypt request"))

	return &reporter{
		meter: meter,
	}
}

func (r *reporter) ReportEncryptCountMetric(ctx context.Context) {
	labels := []attribute.KeyValue{
		attribute.String(osTypeKey, runtimeOS),
	}
	encryptTotal.Add(ctx, 1, labels...)
}

func (r *reporter) ReportDecryptCountMetric(ctx context.Context) {
	labels := []attribute.KeyValue{
		attribute.String(osTypeKey, runtimeOS),
	}
	encryptTotal.Add(ctx, 1, labels...)
}

func (r *reporter) ReportEncryptErrorCountMetric(ctx context.Context, errorType string) {
	labels := []attribute.KeyValue{
		attribute.String(errorTypeKey, errorType),
		attribute.String(osTypeKey, runtimeOS),
	}
	encryptErrorTotal.Add(ctx, 1, labels...)
}

func (r *reporter) ReportDecryptErrorCountMetric(ctx context.Context, errorType string) {
	labels := []attribute.KeyValue{
		attribute.String(errorTypeKey, errorType),
		attribute.String(osTypeKey, runtimeOS),
	}
	encryptErrorTotal.Add(ctx, 1, labels...)
}

func (r *reporter) ReportEncryptDurationMetric(ctx context.Context, duration float64) {
	r.meter.RecordBatch(ctx,
		[]attribute.KeyValue{
			attribute.String(osTypeKey, runtimeOS),
		},
		encryptDuration.Measurement(duration),
	)
}

func (r *reporter) ReportDecryptDurationMetric(ctx context.Context, duration float64) {
	r.meter.RecordBatch(ctx,
		[]attribute.KeyValue{
			attribute.String(osTypeKey, runtimeOS),
		},
		decryptDuration.Measurement(duration),
	)
}
