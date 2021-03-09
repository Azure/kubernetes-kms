ARG BASEIMAGE="gcr.io/distroless/static:nonroot-amd64"
FROM $BASEIMAGE

COPY ./_output/kubernetes-kms /bin/

ENTRYPOINT [ "/bin/kubernetes-kms" ]
