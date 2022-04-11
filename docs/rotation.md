# Rotating KMS key

This guide demonstrates steps required to update your cluster to use a new KMS key for encryption.

> NOTE: Ensure to read the Kubernetes documentation on [Rotating a decryption key](https://kubernetes.io/docs/tasks/administer-cluster/encrypt-data/#rotating-a-decryption-key) before proceeding with the guide.

### 1. Generate a new key or rotate the existing key

* If this is a new key in a different keyvault, then give the cluster identity permissions to access the keys in keyvault. Refer to [doc](./manual-install.md#2-give-the-cluster-identity-permissions-to-access-the-keys-in-keyvault) for details.
* If this is a new version of the same key that's already being used, then proceed to the next step.

### 2. Deploy another instance of KMS plugin with new key

To rotate the encrypt/decrypt key in the cluster, you'll need to run 2 kms plugin pods simultaneously listening on different unix sockets before making the transition.

For all Kubernetes master nodes, add the static pod manifest to `/etc/kubernetes/manifests`

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: azure-kms-provider-2
  namespace: kube-system
  labels:
    tier: control-plane
    component: azure-kms-provider
spec:
  priorityClassName: system-node-critical
  hostNetwork: true
  containers:
    - name: azure-kms-provider
      image: mcr.microsoft.com/oss/azure/kms/keyvault:v0.3.0
      imagePullPolicy: IfNotPresent
      args:
      - --listen-addr=unix:///opt/azurekms2.socket            # unix:///opt/azurekms.socket is used by the primary kms plugin pod. So use a different listen address here for the new kms plugin pod.
      - --keyvault-name=${KV_NAME}                            # [REQUIRED] Name of the keyvault
      - --key-name=${KEY_NAME}                                # [REQUIRED] Name of the keyvault key used for encrypt/decrypt
      - --key-version=${KEY_VERSION}                          # [REQUIRED] Version of the key to use
      - --log-format-json=false                               # [OPTIONAL] Set log formatter to json. Default is false.
      - --healthz-port=8788                                   # The port used here should be different than the one used by the primary kms plugin pod.
      - --healthz-path=/healthz                               # [OPTIONAL] path for health check. Default is /healthz
      - --healthz-timeout=20s                                 # [OPTIONAL] RPC timeout for health check. Default is 20s
      - --managed-hsm=false                                   # [OPTIONAL] Use Azure Key Vault managed HSM. Default is false.
      - -v=5
      securityContext:
        allowPrivilegeEscalation: false
        capabilities:
          drop:
          - ALL
        readOnlyRootFilesystem: true
        runAsUser: 0
      ports:
        - containerPort: 8788                                 # Must match the value defined in --healthz-port
          protocol: TCP
      livenessProbe:
        httpGet:
          path: /healthz                                      # Must match the value defined in --healthz-path
          port: 8788                                          # Must match the value defined in --healthz-port
        failureThreshold: 2
        periodSeconds: 10
      resources:
        requests:
          cpu: 100m
          memory: 128Mi
        limits:
          cpu: "4"
          memory: 2Gi
      volumeMounts:
        - name: etc-kubernetes
          mountPath: /etc/kubernetes
        - name: etc-ssl
          mountPath: /etc/ssl
          readOnly: true
        - name: sock
          mountPath: /opt
  volumes:
    - name: etc-kubernetes
      hostPath:
        path: /etc/kubernetes
    - name: etc-ssl
      hostPath:
        path: /etc/ssl
    - name: sock
      hostPath:
        path: /opt
  nodeSelector:
    kubernetes.io/os: linux
```

View logs from the kms pod:

```bash
kubectl logs -l component=azure-kms-provider -n kube-system

I0219 17:35:33.608840       1 main.go:60] "Starting KeyManagementServiceServer service" version="v0.0.11" buildDate="2021-02-19-17:33"
I0219 17:35:33.609090       1 azure_config.go:27] populating AzureConfig from /etc/kubernetes/azure.json
I0219 17:35:33.609420       1 auth.go:66] "azure: using client_id+client_secret to retrieve access token" clientID="9a7a##### REDACTED #####bb26" clientSecret="23T.##### REDACTED #####vw-r"
I0219 17:35:33.609568       1 keyvault.go:66] "using kms key for encrypt/decrypt" vaultName="k8skmskv" keyName="key1" keyVersion="5cdf48ea6bb9456ebf637e1130b7751a"
I0219 17:35:33.609897       1 main.go:86] Listening for connections on address: /opt/azurekms2.socket
...
```

### 3. Add the new provider to encryption configuration in `/etc/kubernetes/manifests/encryptionconfig.yaml`

```yaml
kind: EncryptionConfiguration
apiVersion: apiserver.config.k8s.io/v1
resources:
  - resources:                                            # List of kubernetes resources that will be encrypted in etcd using the KMS plugin
      - secrets
    providers:
      - kms:
          name: azurekmsprovider
          endpoint: unix:///opt/azurekms.socket           # This endpoint must match the value defined in --listen-addr for the KMS plugin using old key
          cachesize: 1000
      - kms:
          name: azurekmsprovider2
          endpoint: unix:///opt/azurekms2.socket          # This endpoint must match the value defined in --listen-addr for the KMS plugin using new key
          cachesize: 1000
```

### 4. Restart all `kube-apiserver`

* Proceed to the next step if using a single `kube-apiserver`
* If using multi-master, restart the `kube-apiserver` to ensure each server can still decrypt using the new key in the encryption config.
* To validate the decryption still works, run `kubectl get secret <secret name> -o yaml` with one of the existing secrets to confirm the data is returned and is valid.

### 5. Switch the order of provider in the encryption config

```yaml
kind: EncryptionConfiguration
apiVersion: apiserver.config.k8s.io/v1
resources:
  - resources:                                            # List of kubernetes resources that will be encrypted in etcd using the KMS plugin
      - secrets
    providers:
      # kms provider with new key
      - kms:
          name: azurekmsprovider2
          endpoint: unix:///opt/azurekms2.socket          # This endpoint must match the value defined in --listen-addr for the KMS plugin using new key
          cachesize: 1000
      # kms provider with old key
      - kms:
          name: azurekmsprovider
          endpoint: unix:///opt/azurekms.socket           # This endpoint must match the value defined in --listen-addr for the KMS plugin using old key
          cachesize: 1000
```

### 6. Restart all `kube-apiserver` again

Refer to [step 4](#4-restart-all-kube-apiserver) to again restart the `kube-apiserver` for the encryption config changes to take effect.

### 7. Decrypt and re-encrypt existing secrets with new key

Since secrets are encrypted on write, performing an update on a secret will encrypt that content.

Run `kubectl get secrets --all-namespaces -o json | kubectl replace -f -` to encrypt all existing secrets with the new key.

> NOTE: For larger clusters, you may wish to subdivide the secrets by namespace or script an update.

#### How does this work?

The first provider in the encryption configuration is used for new encrypt calls. For decrypt, all existing kms providers in encryption configuration will be tried until one of the decrypt call succeeds.

### 8. Remove the old provider from encryption configuration

Now that all the secrets have been re-encrypted with the new key, we can safely remove the old kms provider from the encryption configuration.

```yaml
kind: EncryptionConfiguration
apiVersion: apiserver.config.k8s.io/v1
resources:
  - resources:                                            # List of kubernetes resources that will be encrypted in etcd using the KMS plugin
      - secrets
    providers:
      # kms provider with new key
      - kms:
          name: azurekmsprovider2
          endpoint: unix:///opt/azurekms2.socket          # This endpoint must match the value defined in --listen-addr for the KMS plugin using new key
          cachesize: 1000
```
