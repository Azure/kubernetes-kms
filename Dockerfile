FROM alpine:3.5
WORKDIR /bin

ADD ./k8s-azure-kms /bin/k8s-azure-kms

CMD ["./k8s-azure-kms"] 