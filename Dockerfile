FROM us.gcr.io/k8s-artifacts-prod/build-image/debian-base-amd64:buster-v1.3.0
COPY ./_output/kubernetes-kms /bin/

ENTRYPOINT [ "/bin/kubernetes-kms" ]
