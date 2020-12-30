# Meerkat

Meerkat is a Kubernetes Operator that facilitates the deployment of OpenVPN in a Kubernetes
cluster. By leveraging [Hashicorp Vault](https://www.vaultproject.io/), Meerkat securely manages
the underlying PKI.

## Features

Meerkat revolves around two CRDs, namely `OvpnServer` and `OvpnClient`. There may exist arbitrarily
many servers while clients are always associated with a single server. These two CRDs give rise to
the following features:

- Generation of shared secrets for TLS Auth
- Creation of a PKI for each server independently with secure private key
- Dynamic OVPN server configuration
- Rendering of `ovpn` client files for each client
- Revocation of client certificates as an `OvpnClient` is deleted

## Usage

This section gives a very brief overview of how Meerkat may be installed in your cluster.

### Prerequisites

In order to use Meerkat, you must have access to a Vault instance. It requires the following:

- Kubernetes Auth has to be enabled and a role for Meerkat has to be defined
- A service account must be configured with a policy to manage PKIs at a specified path (and its
  subpaths).

### Operator Deployment

Then, you can deploy the operator using Helm:

```bash
helm repo add borchero https://charts.borchero.com
helm install meerkat borchero/meerkat \
    --set rbac.serviceAccountName=${SERVICE_ACCOUNT_NAME} \
    --set vault.auth.config.role=${KUBERNETES_ROLE} \
    --set vault.pkiPath=${PKI_PATH}
```

You can also leave all of these fields blank and they choose sensitive defaults. Consult the
[values file](./deploy/values.yaml) for further details.

### Custom Resources

Once the operator is running, you can install the custom resources, creating a server and your
clients. Have a look at the [example manifests](./tests/manifests).

Once a client is created, there exists a secret with the client's name, containing the client's
OVPN certificate. It can be retrieved by using `kubectl`:

```bash
kubectl get secret <SECRET_NAME> -o json | jq -r '.data."certificate.ovpn"' | base64 -d
```

## License

Meerkat is licensed under the [MIT License](./LICENSE).
