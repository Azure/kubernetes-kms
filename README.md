# k8s-azure-kms #

Kubernetes KMS plugin for Azure Key Vault - Enable encryption at rest of Kubernetes data in etcd using Azure Key Vault

From [Kubernetes doc](https://kubernetes.io/docs/tasks/administer-cluster/encrypt-data/#providers)
[KMS plugin is]
> The recommended choice for using a third party tool for key management. Simplifies key rotation, with a new data encryption key (DEK) generated for each encryption, and key encryption key (KEK) rotation controlled by the user. 

⚠️ NOTE: Currently KMS plugin for Kubernetes does not support KMS key rotation scenarios. This means if you choose to create a new key version in KMS, the cluster will fail to decrypt as it won't match the key used for encryption at the time the cluster was created.

💡 NOTE: To store and manage access to application secrets outside of Kubernetes, use [Kubernetes Key Vault FlexVolume](https://github.com/Azure/kubernetes-keyvault-flexvol).

## Features and Concepts ##
* Use a key in Key Vault for etcd encryption
* Generate a HSM-protected (Hardware Security Modules) key
* Bring your own keys
* Secrets/keys/certs are still stored in etcd, managed as part of Kubernetes
* Restrict access using K8s concepts: RBAC, Service Accounts, namespaces

## How to use ##

### Prerequisites: ### 

💡 Make sure you have a Kubernetes cluster v1.10+, minimum version required that supports KMS provider.

### 🎁 aks-engine ###
We have added this feature to aks-engine so that you do not have to worry about any of the manual steps to set this up. Follow this [doc](https://github.com/Azure/aks-engine/blob/master/docs/kubernetes/features.md#azure-key-vault-data-encryption) and this [api model json](https://github.com/Azure/aks-engine/blob/master/examples/kubernetes-config/kubernetes-keyvault-encryption.json) to create your own Kubernetes cluster with Azure Key Vault data encryption. Once the cluster is created, you will see an Azure Key Vault in the same resource group as your cluster, and a new key in the Azure Key Vault.

### Azure Kubernetes Service (AKS) ###
AKS does encrypt secrets-at-rest, but it doesn’t use kubernetes-kms to do this and it doesn’t allow users to bring their own keys.  Keys are managed by AKS.

## Verifying that Data is Encrypted ##

Now that your cluster has `--experimental-encryption-provider-config` turned on, it will use the information provided to encrypt the data in etcd. 

1. Create a new secret

```bash
kubectl create secret generic secret1 -n default --from-literal=mykey=mydata
```

2. Using etcdctl, read the secret out of the etcd:

```bash
sudo ETCDCTL_API=3 etcdctl --cacert=/etc/kubernetes/certs/ca.crt --cert=/etc/kubernetes/certs/etcdclient.crt --key=/etc/kubernetes/certs/etcdclient.key get /registry/secrets/default/secret1
```

3. Verify the stored secret is prefixed with `k8s:enc:kms:v1:azurekmsprovider` which indicates the azure kms provider has encrypted the resulting data.

4. Verify the secret is correctly decrypted when retrieved via the API:

```bash
kubectl get secrets secret1 -o yaml
```
the output should match `mykey: bXlkYXRh`, which is the encoded data of `mydata`. 


# Contributing

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.microsoft.com.

When you submit a pull request, a CLA-bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., label, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.



