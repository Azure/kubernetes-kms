module github.com/Azure/kubernetes-kms

go 1.16

require (
	github.com/Azure/azure-sdk-for-go v48.2.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.1
	github.com/Azure/go-autorest/autorest/adal v0.9.5
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.0 // indirect
	github.com/go-logr/logr v0.3.0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	go.opentelemetry.io/otel v0.20.0
	go.opentelemetry.io/otel/exporters/metric/prometheus v0.20.0
	go.opentelemetry.io/otel/metric v0.20.0
	go.uber.org/zap v1.15.0 // indirect
	golang.org/x/crypto v0.0.0-20201002170205-7f63de1d35b0
	golang.org/x/mod v0.3.0 // indirect
	golang.org/x/net v0.0.0-20201110031124-69a78807bb2b
	golang.org/x/tools v0.0.0-20200616133436-c1934b75d054 // indirect
	google.golang.org/grpc v1.37.0
	gopkg.in/yaml.v2 v2.3.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
	k8s.io/apiserver v0.20.1
	k8s.io/component-base v0.20.2
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.4.0
)
