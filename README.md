# OpenProject Operator

The **OpenProject Operator** automates the creation and scheduling of tickets in an OpenProject server using Kubernetes Custom Resources. Designed for DevOps or engineering teams managing recurring work items, the operator creates and tracks tickets on a schedule you define through CRDs.

## ✨ Features

- Declarative ticket creation using Kubernetes CRDs
- Scheduled ticket generation via cron expressions
- Helm-installable with pre-install CRD verification
- Dynamic OpenProject server configuration via `ServerConfig`
- Intelligent status tracking with last and next run timestamps

## 📦 CRDs

The operator defines two Custom Resource Definitions:

- `ServerConfig` – Defines credentials and endpoint info for a given OpenProject instance.
- `WorkPackages` – Represents a repeatable ticket definition with scheduling and linkage to an OpenProject project and type.

### Example WorkPackage

```yaml
apiVersion: openproject.org/v1alpha1
kind: WorkPackages
metadata:
  name: ticket-schedule-1
  namespace: default
spec:
  serverConfigRef:
    name: dev-server-01
  subject: "Helm Default Ticket"
  description: |
    This ticket was created by the Kubernetes OpenProject operator.
    ## Markdown Header
    * markdown bullet 
    * markdown bullet

    `extra credit code block`

    ### Markdown Header 3
    1. one
    2. two
    3. three
  schedule: "*/2 * * * *"
  projectID: 4
  typeID: 6
  epicID: 338
```

## 🚀 Getting Started

### Prerequisites

- Go 1.22+
- Docker 20.10+
- Kubernetes 1.24+
- `kubectl` and `helm` installed and configured

---

### Build and Push Docker Image

```sh
make docker-build docker-push IMG=shrapk2/openproject-operator:<tag>
```

### Install CRDs

```sh
make install
```

### Deploy the Operator

```sh
make deploy IMG=shrapk2/openproject-operator:<tag>
```

---

## 🧠 Helm Chart Deployment

The Helm charts are located in the `charts/` directory. You can install the operator and its required resources using:

```sh
helm repo add openproject-operator https://shrapk2.github.io/openproject-operator
helm repo update
helm search repo openproject-operator
#
helm install openproject-operator openproject-operator/openproject-operator --set image.repository=shrapk2/openproject-operator --set image.tag=<tag>
```

Then deploy one or more `WorkPackages` using the companion Helm chart:

```sh
helm install dev-server-config openproject-operator/openproject-serverconfig
helm install daily-workpackage-1 openproject-operator/openproject-workpackage
```

### CRD Check Hook

The Helm chart includes a pre-install hook to validate the presence of required CRDs (`ServerConfig`, `WorkPackages`) and will fail gracefully with an appropriate message if they are missing.

---

## 🐛 Debugging & Logging

Set the following environment variable to enable verbose debugging:

```sh
DEBUG=true make run
```

In production, default logs are scoped with minimal context:

- Name
- Schedule
- Status

When `DEBUG` is enabled, the logs include payloads, headers, and reconciliation internals.

---

## 🧼 Uninstalling

```sh
make undeploy
make uninstall
```

Or via Helm:

```sh
helm uninstall openproject-operator
```

---

## 💡 Contributing

Feature requests, PRs, and feedback are welcome! Please:

- Fork the repository
- Create a feature branch
- Run `make test` and `make vet`
- Submit a PR with context

---

## 📄 License

Apache 2.0 — see [LICENSE](./LICENSE)

---

## 📚 Resources

- [Kubebuilder Book](https://book.kubebuilder.io/)
- [OpenProject REST API](https://docs.openproject.org/api/)
