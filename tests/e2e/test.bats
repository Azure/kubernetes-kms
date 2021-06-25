#!/usr/bin/env bats

load helpers

WAIT_TIME=120
SLEEP_TIME=1

if [ ${IS_SOAK_TEST} = true ]; then
    export ETCD_CA_CERT=/etc/kubernetes/certs/ca.crt
    export ETCD_CERT=/etc/kubernetes/certs/etcdclient.crt
    export ETCD_KEY=/etc/kubernetes/certs/etcdclient.key
else
    export ETCD_CA_CERT=/etc/kubernetes/pki/etcd/ca.crt
    export ETCD_CERT=/etc/kubernetes/pki/etcd/server.crt
    export ETCD_KEY=/etc/kubernetes/pki/etcd/server.key
fi

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
    if [ ${IS_SOAK_TEST} = true ]; then
      local node_name=$(kubectl get nodes -l kubernetes.azure.com/role=master -o jsonpath="{.items[0].metadata.name}")
      run kubectl node-shell ${node_name} -- sh -c "ETCDCTL_API=3 etcdctl --cacert=${ETCD_CA_CERT} --cert=${ETCD_CERT} --key=${ETCD_KEY} get /registry/secrets/default/secret1"
      assert_match "k8s:enc:kms:v1:azurekmsprovider" "${output}"
      assert_success
    else
      local pod_name=$(kubectl get pod -n kube-system -l component=etcd -o jsonpath="{.items[0].metadata.name}")
      run kubectl exec ${pod_name} -n kube-system -- etcdctl --cacert=${ETCD_CA_CERT} --cert=${ETCD_CERT} --key=${ETCD_KEY} get /registry/secrets/default/secret1
      assert_match "k8s:enc:kms:v1:azurekmsprovider" "${output}"
      assert_success
    fi
}

@test "check if metrics endpoint works" {
    local randomString=$(openssl rand -hex 5)
    kubectl run curl-${randomString} --image=curlimages/curl:7.75.0 --labels="test=metrics_test" -- tail -f /dev/null
    kubectl wait --for=condition=Ready --timeout=60s pod curl-${randomString}

    local pod_ip=$(kubectl get pod -n kube-system -l component=azure-kms-provider -o jsonpath="{.items[0].status.podIP}")
    run kubectl exec curl-${randomString} -- curl http://${pod_ip}:8095/metrics
    assert_match "kms_request_bucket" "${output}"
    assert_success
}

@test "check healthz for kms plugin" {
    local randomString=$(openssl rand -hex 5)
    kubectl run curl-${randomString} --image=curlimages/curl:7.75.0 --labels="test=healthz_test" -- tail -f /dev/null
    kubectl wait --for=condition=Ready --timeout=60s pod curl

    local pod_ip=$(kubectl get pod -n kube-system -l component=azure-kms-provider -o jsonpath="{.items[0].status.podIP}")
    result=$(kubectl exec curl-${randomString} -- curl http://${pod_ip}:8787/healthz)
    [[ "${result//$'\r'}" == "ok" ]]

    result=$(kubectl exec curl-${randomString} -- curl http://${pod_ip}:8787/healthz -o /dev/null -w '%{http_code}\n' -s)
    [[ "${result//$'\r'}" == "200" ]]

    # cleanup
    run kubectl delete pod curl --force --grace-period 0
}

teardown_file() {
    # cleanup
    run kubectl delete secret secret1 -n default

    run kubectl delete pod -l test=metrics_test --force --grace-period 0
    run kubectl delete pod -l test=healthz_test --force --grace-period 0
}
