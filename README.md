# KMS Plugin for Key Vault

[![CircleCI](https://circleci.com/gh/Azure/kubernetes-kms/tree/master.svg?style=svg)](https://circleci.com/gh/Azure/kubernetes-kms/tree/master)

Enables encryption at rest of your Kubernetes data in etcd using Azure Key Vault.

From the Kubernetes documentation on [Encrypting Secret Data at Rest]:

> [KMS Plugin for Key Vault is] the recommended choice for using a third party tool for key management. Simplifies key rotation, with a new data encryption key (DEK) generated for each encryption, and key encryption key (KEK) rotation controlled by the user.

⚠️ **NOTE**: Currently, KMS plugin for Key Vault does not support key rotation. If you create a new key version in KMS, decryption will fail since it won't match the key used for encryption when the cluster was created.

💡 **NOTE**: To integrate your application secrets from a key management system outside of Kubernetes, use [Key Vault FlexVolume].

## Features

* Use a key in Key Vault for etcd encryption
* Generate keys protected by a Hardware Security Module (HSM)
* Bring your own keys
* Store secrets, keys, and certs in etcd, but manage them as part of Kubernetes
* Restrict access using Kubernetes core concepts: RBAC, Service Accounts, and namespaces

## Getting Started

### Prerequisites

💡 Make sure you have a Kubernetes cluster version 1.10 or later, the minimum version that is supported by KMS Plugin for Key Vault.

### 🎁 aks-engine

[AKS Engine] creates customized Kubernetes clusters on Azure.

Follow the AKS Engine documentation about [Azure Key Vault Data Encryption] and refer to the [example cluster configuration] to create a Kubernetes cluster with KMS Plugin for Key Vault automatically configured. Once the cluster is running, there will be an Azure Key Vault containing a new key in the same resource group as the cluster.

### Azure Kubernetes Service (AKS)

Azure Kubernetes Service ([AKS]) creates managed, supported Kubernetes clusters on Azure.

AKS does encrypt secrets at rest, but keys are managed by the service and users cannot bring their own.

## Verifying that Data is Encrypted

Now that your cluster has `--experimental-encryption-provider-config` turned on, it will encrypt the data in etcd. Let's verify that is working:

1. Create a new secret:

    ```bash
    kubectl create secret generic secret1 -n default --from-literal=mykey=mydata
    ```

2. Using `etcdctl`, read the secret from etcd:

    ```bash
    sudo ETCDCTL_API=3 etcdctl --cacert=/etc/kubernetes/certs/ca.crt --cert=/etc/kubernetes/certs/etcdclient.crt --key=/etc/kubernetes/certs/etcdclient.key get /registry/secrets/default/secret1
    ```

3. Check that the stored secret is prefixed with `k8s:enc:kms:v1:azurekmsprovider`. This indicates the Azure KMS provider has encrypted the data.

4. Verify the secret is decrypted correctly when retrieved via the Kubernetes API:

    ```bash
    kubectl get secrets secret1 -o yaml
    ```

    The output should match `mykey: bXlkYXRh`, which is the encoded data of `mydata`.

## Contributing

The KMS Plugin for Key Vault project welcomes contributions and suggestions. Please see [CONTRIBUTING](CONTRIBUTING.md) for details.

## Code of conduct

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/). For more information, see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq) or contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.


[AKS]: https://azure.microsoft.com/services/kubernetes-service/
[AKS Engine]: https://github.com/Azure/aks-engine
[Azure Key Vault Data Encryption]: https://github.com/Azure/aks-engine/blob/master/docs/topics/features.md#azure-key-vault-data-encryption
[Encrypting Secret Data at Rest]: https://kubernetes.io/docs/tasks/administer-cluster/encrypt-data/#providers
[example cluster configuration]: https://github.com/Azure/aks-engine/blob/master/examples/kubernetes-config/kubernetes-keyvault-encryption.json
[Key Vault FlexVolume]: https://github.com/Azure/kubernetes-keyvault-flexvol
