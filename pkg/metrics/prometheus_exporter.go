package metrics

import (
	"fmt"
	"net/http"
	"time"

	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"monis.app/mlog"
)

const (
	metricsEndpoint = "metrics"
)

func initPrometheusExporter(metricsAddress string) error {
	registry := promclient.NewRegistry()
	exporter, err := prometheus.New(
		prometheus.WithRegisterer(registry))
	if err != nil {
		return err
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
		sdkmetric.WithView(sdkmetric.NewView(
			sdkmetric.Instrument{Kind: sdkmetric.InstrumentKindHistogram},
			sdkmetric.Stream{
				Aggregation: sdkmetric.AggregationExplicitBucketHistogram{
					Boundaries: promclient.ExponentialBucketsRange(0.1, 2, 11),
				},
			},
		)),
	)
	otel.SetMeterProvider(mp)

	http.Handle(fmt.Sprintf("/%s", metricsEndpoint), promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	go func() {
		server := &http.Server{
			Addr:              fmt.Sprintf(":%s", metricsAddress),
			ReadHeaderTimeout: 5 * time.Second,
		}
		if err := server.ListenAndServe(); err != nil {
			mlog.Fatal(err, "failed to register prometheus endpoint", "metricsAddress", metricsAddress)
		}
	}()
	mlog.Always("Prometheus metrics server running", "address", metricsAddress)

	return nil
}
