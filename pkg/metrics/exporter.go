package metrics

import (
	"fmt"
	"strings"

	"monis.app/mlog"
)

const (
	prometheusExporter = "prometheus"
)

// InitMetricsExporter initializes new exporter.
func InitMetricsExporter(metricsBackend, metricsAddress string) error {
	exporter := strings.ToLower(metricsBackend)
	mlog.Always("metrics backend", "exporter", exporter)

	switch exporter {
	// Prometheus is the only exporter supported for now
	case prometheusExporter:
		return initPrometheusExporter(metricsAddress)
	default:
		return fmt.Errorf("unsupported metrics backend %v", metricsBackend)
	}
}
