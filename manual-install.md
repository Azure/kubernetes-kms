# 🛠 Manual Configurations #

1. Assume you already have a Kubernetes cluster. Create a Key Vault in the same resource group as your Kubernetes cluster. Then update the key vault's access policy with the service principal used to create your Kubernetes cluster:

```bash
az keyvault create -n k8skv -g mykubernetesrg
az keyvault set-policy -n k8skv --key-permissions create decrypt encrypt get list --spn <YOUR SPN CLIENT ID>
```
If you do not have a service principal, please refer to this [doc](https://docs.microsoft.com/en-us/cli/azure/create-an-azure-service-principal-azure-cli?view=azure-cli-latest).

From all Kubernetes master nodes:

2. Update `/etc/kubernetes/azure.json` to add the following configurations:

```json
...
    "providerVaultName": "<NAME OF THE KEY VAULT CREATED IN PREVIOUS STEP>",
    "providerKeyName": "<NAME OF THE KEY>",
    "providerKeyVersion": ""

```
* `providerVaultName`: name of the key vault you have created in the same resource group as your k8s cluster. If the provided key vault is not found, service will return an error.
* `providerKeyName`: name of the key created in azure key vault. If the provided key is not found, the gRPC service will create a key `providerKeyName` in the `providerVaultName` key vault.
* `providerKeyVersion`: [OPTIONAL] key version of the key created in azure key vault. If the provided key version is not found, service will return an error. If not provided, service will use a key version from key `providerKeyName` in the `providerVaultName` key vault.

3. Create a systemd unit file `/etc/systemd/system/kms.service`

```
[Unit]
Description=azurekms
Requires=docker.service
After=network-online.target

[Service]
Type=simple
Restart=always
TimeoutStartSec=0
ExecStart=/usr/bin/docker run \
  --net=host \
  --volume=/opt:/opt \
  --volume=/etc/kubernetes:/etc/kubernetes \
  --volume=/etc/ssl/certs/ca-certificates.crt:/etc/ssl/certs/ca-certificates.crt \
  --volume=/var/lib/waagent:/var/lib/waagent \
  microsoft/k8s-azure-kms:latest

[Install]
WantedBy=multi-user.target
```
Enable the service and start it:

```bash
systemctl enable kms && systemctl start kms
```
Verify the gRPC service is running here:

```bash
ls /opt/azu*
/opt/azurekms.socket
```

View logs from the running docker container:
```bash
$ docker ps | grep k8s-azure-kms
c562d0a360dc    microsoft/k8s-azure-kms:latest  ".k8s-azure-kms"    1 min ago   Up 1 min    nostalgic_knuth

$ docker logs c562d0a360dc
/opt/azurekms.socket	
2018/02/26 22:52:26 KeyManagementServiceServer service starting...	
2018/02/26 22:52:26 KeyManagementServiceServer service started successfully.
...
```

4. Update the kubelet.service unit file `/etc/systemd/system/kubelet.service` by adding `Requires=kms.service` as shown:

```
[Unit]
Description=Kubelet
ConditionPathExists=/usr/local/bin/kubelet
Requires=kms.service

```
Reload the service and restart it:

```bash
systemctl daemon-reload && systemctl restart kubelet
```

5. Create `/etc/kubernetes/manifests/encryptionconfig.yaml`

```yaml
kind: EncryptionConfig
apiVersion: v1
resources:
  - resources:
    - secrets
    providers:
    - kms:
        name: azurekmsprovider
        endpoint: unix:///opt/azurekms.socket
        cachesize: 0
    - identity: {}
```

6. Modify `/etc/kubernetes/manifests/kube-apiserver.yaml` 
Add the following flag:

```yaml
--experimental-encryption-provider-config=/etc/kubernetes/manifests/encryptionconfig.yaml
```  
Mount `/opt` to access the socket:

```yaml
...
 volumeMounts:
        - name: "sock"
          mountPath: "/opt"
...
 volumes:
    - name: "sock"
      hostPath:
        path: "/opt"

```

7. Restart apiserver