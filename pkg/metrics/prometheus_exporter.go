package metrics

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/otel/exporters/metric/prometheus"
	"k8s.io/klog/v2"
)

const (
	metricsEndpoint = "metrics"
)

func initPrometheusExporter(metricsAddress string) error {
	exporter, err := prometheus.InstallNewPipeline(prometheus.Config{
		DefaultHistogramBoundaries: []float64{
			0.1, 0.2, 0.3, 0.4, 0.5, 1, 1.5, 2, 2.5, 3.0, 5.0, 10.0, 15.0, 30.0,
		}},
	)
	if err != nil {
		return fmt.Errorf("failed to register prometheus exporter: %v", err)
	}

	http.HandleFunc(fmt.Sprintf("/%s", metricsEndpoint), exporter.ServeHTTP)
	go func() {
		server := &http.Server{
			Addr:              fmt.Sprintf(":%s", metricsAddress),
			ReadHeaderTimeout: 5 * time.Second,
		}
		if err := server.ListenAndServe(); err != nil {
			klog.ErrorS(err, "failed to register prometheus endpoint", "metricsAddress", metricsAddress)
			os.Exit(1)
		}
	}()

	klog.InfoS("Prometheus metrics server running", "address", metricsAddress)
	return nil
}
