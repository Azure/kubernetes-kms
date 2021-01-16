FROM us.gcr.io/k8s-artifacts-prod/build-image/debian-base-amd64:buster-v1.2.0
COPY ./_output/kubernetes-kms /bin/
# upgrading apt &libapt-pkg5.0 due to CVE-2020-27350
# upgrading libp11-kit0 due to CVE-2020-29362, CVE-2020-29363 and CVE-2020-29361
RUN apt-mark unhold apt && \
    clean-install ca-certificates apt libapt-pkg5.0 libp11-kit0 wget

ENTRYPOINT [ "/bin/kubernetes-kms" ]
