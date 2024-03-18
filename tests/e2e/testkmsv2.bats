#!/usr/bin/env bats

load helpers

WAIT_TIME=120
SLEEP_TIME=1

export ETCD_CA_CERT=/etc/kubernetes/pki/etcd/ca.crt
export ETCD_CERT=/etc/kubernetes/pki/etcd/server.crt
export ETCD_KEY=/etc/kubernetes/pki/etcd/server.key

setup() {
    # get the initial number of encrypted count
    local metrics=$(kubectl get --raw /metrics)
    expected_encyption_count=$(echo "${metrics}" | grep -oP 'apiserver_envelope_encryption_key_id_hash_total\{[^\}]*transformation_type="to_storage"[^\}]*\}\s+\K\d+')    
}

@test "azure keyvault kms plugin is running" {
    wait_for_process ${WAIT_TIME} ${SLEEP_TIME} "kubectl -n kube-system wait --for=condition=Ready --timeout=60s pod -l component=azure-kms-provider"
}

@test "creating secret resource" {
    run kubectl create secret generic secret1 -n default --from-literal=foo=bar
    let "expected_encyption_count++"
    assert_success
}

@test "read the secret resource test" {
    result=$(kubectl get secret secret1 -o jsonpath='{.data.foo}' | base64 -d)
    [[ "${result//$'\r'}" == "bar" ]]
}

@test "check if secret is encrypted in etcd" {
    local pod_name=$(kubectl get pod -n kube-system -l component=etcd -o jsonpath="{.items[0].metadata.name}")
    run kubectl exec ${pod_name} -n kube-system -- etcdctl --cacert=${ETCD_CA_CERT} --cert=${ETCD_CERT} --key=${ETCD_KEY} get /registry/secrets/default/secret1
    assert_match "k8s:enc:kms:v2:azurekmsprovider" "${output}"
    assert_success
}

@test "check encryption count" {
    # The expected_encryption_count value is set in the setup().
    local metrics=$(kubectl get --raw /metrics)
    encyption_count=$(echo "${metrics}" | grep -oP 'apiserver_envelope_encryption_key_id_hash_total\{[^\}]*transformation_type="to_storage"[^\}]*\}\s+\K\d+')
    [[ "${encyption_count}" == "${expected_encyption_count}" ]]
}

@test "check keyID hash used for encrypt/decrypt" {
    # expected_hash value is computed based on the key used in CI.
    # this needs to be updated when we rotate that key.
    local expected_hash="cbda52be2f8c13d323a3b17c4679118a60b91d29454305e02ee485185b6e386f"
    local metrics=$(kubectl get --raw /metrics)

    hashIDs=$(echo "${metrics}" | grep -oP 'sha256:\K[a-f0-9]+')
    for hash in ${hashIDs}; do
        echo ${hash}
        [[ "${hash}" == "${expected_hash}" ]]
    done
}

@test "check if metrics endpoint works" {
    local curl_pod_name=curl-$(openssl rand -hex 5)
    kubectl run ${curl_pod_name} --image=curlimages/curl:7.75.0 --labels="test=metrics_test" -- tail -f /dev/null
    kubectl wait --for=condition=Ready --timeout=60s pod ${curl_pod_name}

    local pod_ip=$(kubectl get pod -n kube-system -l component=azure-kms-provider -o jsonpath="{.items[0].status.podIP}")
    run kubectl exec ${curl_pod_name} -- curl http://${pod_ip}:8095/metrics
    assert_match "kms_request_bucket" "${output}"
    assert_success
}

@test "check healthz for kms plugin" {
    local curl_pod_name=curl-$(openssl rand -hex 5)
    kubectl run ${curl_pod_name} --image=curlimages/curl:7.75.0 --labels="test=healthz_test" -- tail -f /dev/null
    kubectl wait --for=condition=Ready --timeout=60s pod ${curl_pod_name}

    local pod_ip=$(kubectl get pod -n kube-system -l component=azure-kms-provider -o jsonpath="{.items[0].status.podIP}")
    result=$(kubectl exec ${curl_pod_name} -- curl http://${pod_ip}:8787/healthz)
    [[ "${result//$'\r'}" == "ok" ]]

    result=$(kubectl exec ${curl_pod_name} -- curl http://${pod_ip}:8787/healthz -o /dev/null -w '%{http_code}\n' -s)
    [[ "${result//$'\r'}" == "200" ]]
}

teardown_file() {
    # cleanup
    run kubectl delete secret secret1 -n default

    run kubectl delete pod -l test=metrics_test --force --grace-period 0
    run kubectl delete pod -l test=healthz_test --force --grace-period 0
}
