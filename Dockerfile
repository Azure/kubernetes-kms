FROM us.gcr.io/k8s-artifacts-prod/build-image/debian-base-amd64:buster-v1.4.0
# upgrading libzstd1 due to CVE-2021-24032
RUN clean-install libzstd1
COPY ./_output/kubernetes-kms /bin/

ENTRYPOINT [ "/bin/kubernetes-kms" ]
