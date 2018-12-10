# Developers Guide

This guide explains how to set up your environment for developing the Azure kubernetes kms service.

## Prerequisites

- Go 1.9.0 or later
- dep
- kubectl 1.9 or later
- An Azure account (needed for creating Azure key vault)
- Git
- make

### Structure of the Code

The code for the kubernetes-kms project is organized as follows:

- The built binary is located in root `./kubernetes-kms`
- The `test/` directory contains `client.go`, which creates a connection against the grpc unix service at `/opt/azurekms.socket` then executes client-side API calls against the `KeyManagementService` service. This is used by the CI/CD pipeline.

Go dependencies are managed with [dep](https://github.com/golang/dep) and stored in the
`vendor/` directory.


### Git Conventions

We use Git for our version control system. The `master` branch is the
home of the current development candidate. Releases are tagged.

We accept changes to the code via GitHub Pull Requests (PRs). One
workflow for doing this is as follows:

1. Use `go get` to clone this repository: `go get github.com/Azure/kubernetes-kms`
2. Fork that repository into your GitHub account
3. Add your repository as a remote for `$GOPATH/github.com/Azure/kubernetes-kms`
4. Create a new working branch (`git checkout -b feat/my-feature`) and
   do your work on that branch.
5. When you are ready for us to review, push your branch to GitHub, and
   then open a new pull request with us.

### Build the Code

We use `make` and `Makefile` to build the binary and the Docker image. To start the build process:

1. Run `make build` to build the binary `/kubernetes-kms` for your OS

### Run the Code Locally

To test your code locally:

1. On a linux machine, you can run `sudo ./kubernetes-kms --configFilePath <PATH TO YOUR AZURE.JSON FILE>` to create the gRPC unix domain socket running at `/opt/azurekms.socket`. This will start the gRPC server.
2. Create an Azure resource group, a Key Vault, and update the key vault's access policy with:

```bash
az group create -n mykeyvaultrg -l eastus
az keyvault create -n k8skv -g mykeyvaultrg
az keyvault set-policy -n k8skv --key-permissions create decrypt encrypt get list --spn <YOUR SPN CLIENT ID>
```
If you do not have a service principal, please refer to this [doc](https://docs.microsoft.com/en-us/cli/azure/create-an-azure-service-principal-azure-cli?view=azure-cli-latest).

3. Populate a `azure.json` file locally. The gRPC server will look for this file in the path provided by `configFilePath`. By default, `configFilePath` is set to `etc/kubernetes/azure.json`. 

```json
{
    "tenantId": "<YOUR TENANT ID>",
    "subscriptionId": "<YOUR SUBSCRIPTION ID>",
    "aadClientId": "<YOUR CLIENT ID>",
    "aadClientSecret": "<YOUR CLIENT SECRET>",
    "resourceGroup": "mykeyvaultrg",
    "location": "eastus",
    "providerVaultName": "k8skv",
    "providerKeyName": "mykey"
}
```
4. Test with the gRPC client, run `sudo GOPATH=[YOUR GOPATH] GOCACHE=off go test tests/client/client_test.go`.
5. Test racing condition with the gRPC client, run `sudo GOPATH=[YOUR GOPATH] go test test/client/client_test.go & sudo GOPATH=[YOUR GOPATH] go test test/client/client_test.go &`.

### Build image
1. Run `make build-image` to build the binary `/kubernetes-kms` for linux and Docker image `microsoft/k8s-azure-kms:latest`
