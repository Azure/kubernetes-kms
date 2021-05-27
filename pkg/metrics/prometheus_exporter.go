package metrics

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel/exporters/metric/prometheus"
	"k8s.io/klog"
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
		return fmt.Errorf("Failed to register prometheus exporter - %v", err)
	}

	http.HandleFunc(fmt.Sprintf("/%s", metricsEndpoint), exporter.ServeHTTP)
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%s", metricsAddress), nil)
		klog.Fatalf("Failed to register prometheus endpoint - %v", err)
	}()

	klog.Infof("Prometheus metrics server running on address: %v", metricsAddress)
	return nil
}
