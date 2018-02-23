FROM alpine:3.5
WORKDIR /bin

ADD ./k8s-azure-kms /bin/k8s-azure-kms

ENTRYPOINT ["./k8s-azure-kms"] 