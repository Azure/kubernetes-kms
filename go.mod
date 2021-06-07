module github.com/Azure/kubernetes-kms

go 1.16

require (
	github.com/Azure/azure-sdk-for-go v55.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.18
	github.com/Azure/go-autorest/autorest/adal v0.9.13
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	go.opentelemetry.io/otel v0.20.0
	go.opentelemetry.io/otel/exporters/metric/prometheus v0.20.0
	go.opentelemetry.io/otel/metric v0.20.0
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5
	google.golang.org/grpc v1.38.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/apiserver v0.21.1
	k8s.io/component-base v0.21.1
	k8s.io/klog/v2 v2.9.0
)
