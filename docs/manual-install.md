# 🛠 Manual Configurations #

This guide demonstrates steps required to enable the KMS Plugin for Key Vault in an existing cluster.

### 1. Create a Keyvault

  If you're bringing your own keys, skip this step.

  ```bash
  KEYVAULT_NAME=k8skv
  RG=mykubernetesrg
  LOC=eastus

  # create resource group that'll contain the keyvault instance
  az group create -n $RG -l $LOC
  # create keyvault
  az keyvault create -n $KV_NAME -g $RG
  # create key that will be used for encryption
  az keyvault key create -n k8s --vault-name $KV_NAME --kty RSA --size 2048
  ```

### 2. Give the cluster identity permissions to access the keys in keyvault

  The KMS Plugin uses the cluster service principal or managed identity to access the keyvault instance.

  #### More on authentication methods

  [`/etc/kubernetes/azure.json`](https://kubernetes-sigs.github.io/cloud-provider-azure/install/configs/) is a well-known JSON file in each node that provides the details about which method KMS Plugin uses for access to Keyvault:

  | Authentication method            | `/etc/kubernetes/azure.json` fields used                                                    |
  | -------------------------------- | ------------------------------------------------------------------------------------------- |
  | System-assigned managed identity | `useManagedIdentityExtension: true` and `userAssignedIdentityID:""`                         |
  | User-assigned managed identity   | `useManagedIdentityExtension: true` and `userAssignedIdentityID:"<UserAssignedIdentityID>"` |
  | Service principal (default)      | `aadClientID: "<AADClientID>"` and `aadClientSecret: "<AADClientSecret>"`                   |

  #### Obtaining the ID of the cluster managed identity/service principal

  After your cluster is provisioned, depending on your cluster identity configuration, run one of the following commands to retrieve the **ID** of your managed identity or service principal, which will be used for role assignment to access Keyvault:

  | Cluster configuration              | Command                                                                                                        |
  | ---------------------------------- | -------------------------------------------------------------------------------------------------------------- |
  | AKS cluster with service principal | `az aks show -g <AKSResourceGroup> -n <AKSClusterName> --query servicePrincipalProfile.clientId -otsv`         |
  | AKS cluster with managed identity  | `az aks show -g <AKSResourceGroup> -n <AKSClusterName> --query identityProfile.kubeletidentity.clientId -otsv` |

  Assign the following permissions:

  ```bash
  az keyvault set-policy -n $KEYVAULT_NAME --key-permissions decrypt encrypt --spn <YOUR SPN CLIENT ID>
  ```

### 3. Deploy the KMS Plugin

  For all Kubernetes master nodes, add the static pod manifest to `/etc/kubernetes/manifests`

  ```yaml
  apiVersion: v1
  kind: Pod
  metadata:
    name: azure-kms-provider
    namespace: kube-system
    labels:
      tier: control-plane
      component: azure-kms-provider
  spec:
    priorityClassName: system-node-critical
    hostNetwork: true
    containers:
      - name: azure-kms-provider
        image: mcr.microsoft.com/oss/azure/kms/keyvault:v0.5.0
        imagePullPolicy: IfNotPresent
        args:
          - --listen-addr=unix:///opt/azurekms.socket             # [OPTIONAL] gRPC listen address. Default is unix:///opt/azurekms.socket
          - --keyvault-name=${KV_NAME}                            # [REQUIRED] Name of the keyvault. Must match criteria specified at https://docs.microsoft.com/en-us/azure/key-vault/general/about-keys-secrets-certificates#vault-name-and-object-name
          - --key-name=${KEY_NAME}                                # [REQUIRED] Name of the keyvault key used for encrypt/decrypt
          - --key-version=${KEY_VERSION}                          # [REQUIRED] Version of the key to use
          - --log-format-json=false                               # [OPTIONAL] Set log formatter to json. Default is false.
          - --healthz-port=8787                                   # [OPTIONAL] port for health check. Default is 8787
          - --healthz-path=/healthz                               # [OPTIONAL] path for health check. Default is /healthz
          - --healthz-timeout=20s                                 # [OPTIONAL] RPC timeout for health check. Default is 20s
          - --managed-hsm=false                                   # [OPTIONAL] Use Azure Key Vault managed HSM. Default is false.
          - -v=1
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - ALL
          readOnlyRootFilesystem: true
          runAsUser: 0
        ports:
          - containerPort: 8787                                   # Must match the value defined in --healthz-port
            protocol: TCP
        livenessProbe:
          httpGet:
            path: /healthz                                        # Must match the value defined in --healthz-path
            port: 8787                                            # Must match the value defined in --healthz-port
          failureThreshold: 2
          periodSeconds: 10
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 4
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
  ```

  View logs from the kms pod:

  ```bash
  kubectl logs -l component=azure-kms-provider -n kube-system

  I0219 17:35:33.608840       1 main.go:60] "Starting KeyManagementServiceServer service" version="v0.0.11" buildDate="2021-02-19-17:33"
  I0219 17:35:33.609090       1 azure_config.go:27] populating AzureConfig from /etc/kubernetes/azure.json
  I0219 17:35:33.609420       1 auth.go:66] "azure: using client_id+client_secret to retrieve access token" clientID="9a7a##### REDACTED #####bb26" clientSecret="23T.##### REDACTED #####vw-r"
  I0219 17:35:33.609568       1 keyvault.go:66] "using kms key for encrypt/decrypt" vaultName="k8skmskv" keyName="key1" keyVersion="5cdf48ea6bb9456ebf637e1130b7751a"
  I0219 17:35:33.609897       1 main.go:86] Listening for connections on address: /opt/azurekms.socket
  ...
  ```

### 4. Create encryption configuration

  Create a new encryption configuration file `/etc/kubernetes/manifests/encryptionconfig.yaml` using the appropriate properties for the `kms` provider:

  ```yaml
  kind: EncryptionConfiguration
  apiVersion: apiserver.config.k8s.io/v1
  resources:
    - resources:                                        # List of kubernetes resources that will be encrypted in etcd using the KMS plugin
        - secrets
      providers:
        - kms:
            name: azurekmsprovider
            endpoint: unix:///opt/azurekms.socket       # This endpoint must match the value defined in --listen-addr for the KMS plugin
            cachesize: 1000
        - identity: {}
  ```

  The encryption configuration file needs to be accessible by all the api servers.

### 5. Modify `/etc/kubernetes/kube-apiserver.yaml`

  Add the following flag:

  ```yaml
  --encryption-provider-config=/etc/kubernetes/encryptionconfig.yaml
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

### 6. Restart your API server
