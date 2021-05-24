#!/usr/bin/env bats

load helpers

WAIT_TIME=120
SLEEP_TIME=1
ETCD_CA_CERT=/etc/kubernetes/pki/etcd/ca.crt
ETCD_CERT=/etc/kubernetes/pki/etcd/server.crt
ETCD_KEY=/etc/kubernetes/pki/etcd/server.key

@test "azure keyvault kms plugin is running" {
    wait_for_process ${WAIT_TIME} ${SLEEP_TIME} "kubectl -n kube-system wait --for=condition=Ready --timeout=60s pod -l component=azure-kms-provider"
}

@test "creating secret resource" {
    run kubectl create secret generic secret1 -n default --from-literal=foo=bar
    assert_success
}

@test "read the secret resource test" {
    result=$(kubectl get secret secret1 -o jsonpath='{.data.foo}' | base64 -d)
    [[ "${result//$'\r'}" == "bar" ]]
}

@test "check if secret is encrypted in etcd" {
    local pod_name=$(kubectl get pod -n kube-system -l component=etcd -o jsonpath="{.items[0].metadata.name}")
    run kubectl exec ${pod_name} -n kube-system -- etcdctl --cacert=${ETCD_CA_CERT} --cert=${ETCD_CERT} --key=${ETCD_KEY} get /registry/secrets/default/secret1
    assert_match "k8s:enc:kms:v1:azurekmsprovider" "${output}"
    assert_success

    #cleanup
    run kubectl delete secret secret1 -n default
}

@test "check healthz for kms plugin" {
    kubectl run curl --image=curlimages/curl:7.75.0 -- tail -f /dev/null
    kubectl wait --for=condition=Ready --timeout=60s pod curl

    local pod_ip=$(kubectl get pod -n kube-system -l component=azure-kms-provider -o jsonpath="{.items[0].status.podIP}")
    result=$(kubectl exec curl -- curl http://${pod_ip}:8787/healthz)
    [[ "${result//$'\r'}" == "ok" ]]

    result=$(kubectl exec curl -- curl http://${pod_ip}:8787/healthz -o /dev/null -w '%{http_code}\n' -s)
    [[ "${result//$'\r'}" == "200" ]]

    #cleanup
    run kubectl delete pod curl --force
}
