# Metrics provided by Secrets Store CSI Driver

This project uses [opentelemetry](https://opentelemetry.io/) for reporting metrics. This project is under [active development](https://github.com/open-telemetry/opentelemetry-go#release-schedule)

Prometheus is the only exporter that's currently supported.

## List of metrics provided by the driver

| Metric                          | Description                                                               | Tags                                                                              |
| ------------------------------- | ------------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| total_request                   | Total number of requests                                                  | `status=success OR error`<br>`operation=encrypt OR decrypt`                           |
| duration_seconds                | Distribution of how long it took for an operation                         | `operation=encrypt OR decrypt`                       |


### Sample Metrics output

```shell
# HELP duration_seconds Distribution of how long it took for an operation
# TYPE duration_seconds histogram
duration_seconds_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.1"} 18
duration_seconds_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.2"} 77
duration_seconds_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.3"} 79
duration_seconds_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.4"} 98
duration_seconds_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.5"} 99
duration_seconds_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="1"} 100
duration_seconds_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="1.5"} 100
duration_seconds_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="2"} 100
duration_seconds_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="2.5"} 100
duration_seconds_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="3"} 100
duration_seconds_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="5"} 100
duration_seconds_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="10"} 100
duration_seconds_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="15"} 100
duration_seconds_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="30"} 100
duration_seconds_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="+Inf"} 100
duration_seconds_sum{operation="decrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0"} 18.579140302
duration_seconds_count{operation="decrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0"} 100
duration_seconds_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.1"} 0
duration_seconds_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.2"} 0
duration_seconds_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.3"} 0
duration_seconds_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.4"} 0
duration_seconds_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.5"} 0
duration_seconds_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="1"} 44
duration_seconds_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="1.5"} 90
duration_seconds_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="2"} 95
duration_seconds_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="2.5"} 95
duration_seconds_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="3"} 100
duration_seconds_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="5"} 100
duration_seconds_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="10"} 100
duration_seconds_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="15"} 100
duration_seconds_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="30"} 100
duration_seconds_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="+Inf"} 100
duration_seconds_sum{operation="encrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0"} 115.52740541100002
duration_seconds_count{operation="encrypt",service_name="unknown_service:__debug_bin",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0"} 100
# HELP total_request Total number of requests.
# TYPE total_request counter
total_request{operation="decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0"} 100
total_request{operation="encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0"} 100
```
