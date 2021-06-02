# Metrics provided by KMS plugin for Key Vault

This project uses [opentelemetry](https://opentelemetry.io/) for reporting metrics. Please refer it's status [here](https://github.com/open-telemetry/opentelemetry-go#project-status). Prometheus is the only exporter that's currently supported.

## List of metrics provided by the kms plugin

| Metric                          | Description                                                               | Tags                                                                              |
| ------------------------------- | ------------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| total_request                   | Total number of requests                                                  | `status=success OR error`<br>`operation=encrypt OR decrypt`                           |
| duration_seconds                | Distribution of how long it took for an operation                         | `operation=encrypt OR decrypt`                       |


### Sample Metrics output

```shell
# HELP kms_request Distribution of how long it took for an operation
# TYPE kms_request histogram
kms_request_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.1"} 39
kms_request_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.2"} 77
kms_request_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.3"} 156
kms_request_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.4"} 170
kms_request_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.5"} 180
kms_request_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="1"} 198
kms_request_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="1.5"} 200
kms_request_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="2"} 200
kms_request_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="2.5"} 200
kms_request_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="3"} 200
kms_request_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="5"} 200
kms_request_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="10"} 200
kms_request_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="15"} 200
kms_request_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="30"} 200
kms_request_bucket{operation="decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="+Inf"} 200
kms_request_sum{operation="decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0"} 49.982473866999996
kms_request_count{operation="decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0"} 200
kms_request_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.1"} 0
kms_request_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.2"} 2
kms_request_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.3"} 12
kms_request_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.4"} 36
kms_request_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.5"} 65
kms_request_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="1"} 100
kms_request_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="1.5"} 100
kms_request_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="2"} 137
kms_request_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="2.5"} 168
kms_request_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="3"} 176
kms_request_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="5"} 200
kms_request_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="10"} 200
kms_request_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="15"} 200
kms_request_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="30"} 200
kms_request_bucket{operation="encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="+Inf"} 200
kms_request_sum{operation="encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0"} 271.335309324
kms_request_count{operation="encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0"} 200
kms_request_bucket{operation="grpc_decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.1"} 39
kms_request_bucket{operation="grpc_decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.2"} 77
kms_request_bucket{operation="grpc_decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.3"} 156
kms_request_bucket{operation="grpc_decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.4"} 170
kms_request_bucket{operation="grpc_decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.5"} 180
kms_request_bucket{operation="grpc_decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="1"} 198
kms_request_bucket{operation="grpc_decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="1.5"} 200
kms_request_bucket{operation="grpc_decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="2"} 200
kms_request_bucket{operation="grpc_decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="2.5"} 200
kms_request_bucket{operation="grpc_decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="3"} 200
kms_request_bucket{operation="grpc_decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="5"} 200
kms_request_bucket{operation="grpc_decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="10"} 200
kms_request_bucket{operation="grpc_decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="15"} 200
kms_request_bucket{operation="grpc_decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="30"} 200
kms_request_bucket{operation="grpc_decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="+Inf"} 200
kms_request_sum{operation="grpc_decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0"} 49.993816699999996
kms_request_count{operation="grpc_decrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0"} 200
kms_request_bucket{operation="grpc_encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.1"} 0
kms_request_bucket{operation="grpc_encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.2"} 2
kms_request_bucket{operation="grpc_encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.3"} 12
kms_request_bucket{operation="grpc_encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.4"} 36
kms_request_bucket{operation="grpc_encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.5"} 65
kms_request_bucket{operation="grpc_encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="1"} 100
kms_request_bucket{operation="grpc_encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="1.5"} 100
kms_request_bucket{operation="grpc_encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="2"} 137
kms_request_bucket{operation="grpc_encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="2.5"} 168
kms_request_bucket{operation="grpc_encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="3"} 176
kms_request_bucket{operation="grpc_encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="5"} 200
kms_request_bucket{operation="grpc_encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="10"} 200
kms_request_bucket{operation="grpc_encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="15"} 200
kms_request_bucket{operation="grpc_encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="30"} 200
kms_request_bucket{operation="grpc_encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="+Inf"} 200
kms_request_sum{operation="grpc_encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0"} 271.3962141690001
kms_request_count{operation="grpc_encrypt",service_name="unknown_service:__debug_bin",status="success",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0"} 200
```
