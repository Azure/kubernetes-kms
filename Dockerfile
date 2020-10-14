FROM alpine:3.12
WORKDIR /bin

ADD ./kubernetes-kms /bin/k8s-azure-kms

CMD ["./k8s-azure-kms"]
