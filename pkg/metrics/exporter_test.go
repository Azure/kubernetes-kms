package metrics

import (
	"testing"
)

var (
	metricsAddress = "8096"
)

func TestInitMetricsExporter(t *testing.T) {
	testCases := []struct {
		name           string
		metricsBackend string
		expectedError  bool
	}{
		{
			name:           "With_Prometheus_Backend",
			metricsBackend: "prometheus",
			expectedError:  false,
		},
		{
			name:           "With_Non_Prometheus_Backend",
			metricsBackend: "nonprometheus",
			expectedError:  true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := InitMetricsExporter(testCase.metricsBackend, metricsAddress)

			if testCase.expectedError && err == nil || !testCase.expectedError && err != nil {
				t.Fatalf("expected error: %v, got error: %v", testCase.expectedError, err)
			}
		})
	}
}
