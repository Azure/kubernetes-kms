# Metrics provided by KMS plugin for Key Vault

This project uses [opentelemetry](https://opentelemetry.io/) for reporting metrics. Please refer to it's status [here](https://github.com/open-telemetry/opentelemetry-go#project-status). Prometheus is the only exporter that's currently supported.

## List of metrics provided by the kms plugin

| Metric                          | Description                                                               | Tags                                                                              |
| ------------------------------- | ------------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| kms_request                   | Distribution of how long it took for an operation                                                  | `status=success OR error`<br><br>`operation=encrypt OR decrypt OR grpc_encrypt OR grpc_decrypt`<br><br>`error_message`                           |


### Sample Metrics output

```shell
# HELP kms_request Distribution of how long it took for an operation
# TYPE kms_request histogram
kms_request_bucket{operation="decrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.1"} 18
kms_request_bucket{operation="decrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.13492828476735633"} 18
kms_request_bucket{operation="decrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.18205642030260802"} 18
kms_request_bucket{operation="decrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.24564560522315804"} 18
kms_request_bucket{operation="decrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.3314454017339986"} 18
kms_request_bucket{operation="decrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.4472135954999578"} 18
kms_request_bucket{operation="decrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.6034176336545162"} 18
kms_request_bucket{operation="decrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.8141810630738084"} 18
kms_request_bucket{operation="decrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="1.0985605433061172"} 18
kms_request_bucket{operation="decrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="1.4822688982138947"} 18
kms_request_bucket{operation="decrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="1.9999999999999991"} 18
kms_request_bucket{operation="decrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="+Inf"} 18
kms_request_sum{operation="decrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success"} 1.010053082
kms_request_count{operation="decrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success"} 18
kms_request_bucket{operation="encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.1"} 19
kms_request_bucket{operation="encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.13492828476735633"} 19
kms_request_bucket{operation="encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.18205642030260802"} 19
kms_request_bucket{operation="encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.24564560522315804"} 19
kms_request_bucket{operation="encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.3314454017339986"} 19
kms_request_bucket{operation="encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.4472135954999578"} 19
kms_request_bucket{operation="encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.6034176336545162"} 19
kms_request_bucket{operation="encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.8141810630738084"} 19
kms_request_bucket{operation="encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="1.0985605433061172"} 19
kms_request_bucket{operation="encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="1.4822688982138947"} 19
kms_request_bucket{operation="encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="1.9999999999999991"} 19
kms_request_bucket{operation="encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="+Inf"} 19
kms_request_sum{operation="encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success"} 1.021080768
kms_request_count{operation="encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success"} 19
kms_request_bucket{operation="grpc_encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.1"} 1
kms_request_bucket{operation="grpc_encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.13492828476735633"} 1
kms_request_bucket{operation="grpc_encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.18205642030260802"} 1
kms_request_bucket{operation="grpc_encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.24564560522315804"} 1
kms_request_bucket{operation="grpc_encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.3314454017339986"} 1
kms_request_bucket{operation="grpc_encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.4472135954999578"} 1
kms_request_bucket{operation="grpc_encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.6034176336545162"} 1
kms_request_bucket{operation="grpc_encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.8141810630738084"} 1
kms_request_bucket{operation="grpc_encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="1.0985605433061172"} 1
kms_request_bucket{operation="grpc_encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="1.4822688982138947"} 1
kms_request_bucket{operation="grpc_encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="1.9999999999999991"} 1
kms_request_bucket{operation="grpc_encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="+Inf"} 1
kms_request_sum{operation="grpc_encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success"} 0.053279316
kms_request_count{operation="grpc_encrypt",otel_scope_name="keyvaultkms",otel_scope_version="",status="success"} 1
kms_request_bucket{operation="grpc_status",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.1"} 0
kms_request_bucket{operation="grpc_status",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.13492828476735633"} 11
kms_request_bucket{operation="grpc_status",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.18205642030260802"} 13
kms_request_bucket{operation="grpc_status",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.24564560522315804"} 13
kms_request_bucket{operation="grpc_status",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.3314454017339986"} 13
kms_request_bucket{operation="grpc_status",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.4472135954999578"} 13
kms_request_bucket{operation="grpc_status",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.6034176336545162"} 14
kms_request_bucket{operation="grpc_status",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.8141810630738084"} 14
kms_request_bucket{operation="grpc_status",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="1.0985605433061172"} 14
kms_request_bucket{operation="grpc_status",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="1.4822688982138947"} 14
kms_request_bucket{operation="grpc_status",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="1.9999999999999991"} 14
kms_request_bucket{operation="grpc_status",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="+Inf"} 14
kms_request_sum{operation="grpc_status",otel_scope_name="keyvaultkms",otel_scope_version="",status="success"} 2.1240865880000004
kms_request_count{operation="grpc_status",otel_scope_name="keyvaultkms",otel_scope_version="",status="success"} 14
kms_request_bucket{operation="grpc_version",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.1"} 9
kms_request_bucket{operation="grpc_version",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.13492828476735633"} 9
kms_request_bucket{operation="grpc_version",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.18205642030260802"} 9
kms_request_bucket{operation="grpc_version",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.24564560522315804"} 9
kms_request_bucket{operation="grpc_version",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.3314454017339986"} 9
kms_request_bucket{operation="grpc_version",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.4472135954999578"} 9
kms_request_bucket{operation="grpc_version",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.6034176336545162"} 9
kms_request_bucket{operation="grpc_version",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="0.8141810630738084"} 9
kms_request_bucket{operation="grpc_version",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="1.0985605433061172"} 9
kms_request_bucket{operation="grpc_version",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="1.4822688982138947"} 9
kms_request_bucket{operation="grpc_version",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="1.9999999999999991"} 9
kms_request_bucket{operation="grpc_version",otel_scope_name="keyvaultkms",otel_scope_version="",status="success",le="+Inf"} 9
kms_request_sum{operation="grpc_version",otel_scope_name="keyvaultkms",otel_scope_version="",status="success"} 0.0007254060000000001
kms_request_count{operation="grpc_version",otel_scope_name="keyvaultkms",otel_scope_version="",status="success"} 9
```
