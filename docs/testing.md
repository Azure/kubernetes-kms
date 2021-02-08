# End-to-end testing for KMS Plugin for Keyvault

## Prerequisites

To run tests locally, following components are required:

1. [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
1. [bats](https://bats-core.readthedocs.io/en/latest/installation.html)
1. [kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)

To install the prerequisites, run the following command:

```bash
make e2e-install-prerequisites
```

The E2E test suite extracts runtime configurations through environment variables. Below is a list of environment variables to set before running the E2E test suite.
| Variable      | Description                                                                                         |
| ------------- | --------------------------------------------------------------------------------------------------- |
| CLIENT_ID     | The client ID of your service principal that has `encrypt, decrypt` access to the keyvault key.     |
| CLIENT_SECRET | The client secret of your service principal that has `encrypt, decrypt` access to the keyvault key. |
| TENANT_ID     | The Azure tenant ID.                                                                                |
| KEYVAULT_NAME | The Azure Keyvault name.                                                                            |
| KEY_NAME      | The name of Keyvault key that will be used by the kms plugin.                                       |
| KEY_VERSION   | The version of Keyvault key that will be used by the kms plugin.                                    |

## Running the tests

The e2e tests are run against a [kind](https://kind.sigs.k8s.io/) cluster that's created as part of the test script. The script also creates a local docker registry that's used for test images.

1. Setup cluster, registry and build image:

```bash
make e2e-setup-kind
```

- This creates the local registry
- Builds a kms plugin image with the latest changes and pushes to local registry
- Creates a kind cluster with connectivity to local registry and kms plugin enabled with custom image

1. Run the end-to-end tests:

```bash
make e2e-test
```

1. To delete the kind cluster after running tests:

```bash
make e2e-delete-kind
```
