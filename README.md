# daemonjob

Provides Job CRDs for managing cluster-wide node-local workloads.

## Description

The daemonjob repository provides advanced workload management capabilities that combine the node-targeting behavior of a DaemonSet with the run-to-completion semantics of a Job.

### [DaemonJob](api/v1/daemonjob_types.go)

Executes a task once on every node by dispatching a single broadcast Job that manages per-node worker Jobs.

### [DaemonCronJob](api/v1/daemoncronjob_types.go)

Extends the DaemonJob with scheduling, periodically triggering cluster-wide executions via a dedicated CronJob controller.

### [DaemonCronJobSet](api/v1/daemoncronjobset_types.go)

A high-granularity scheduler that deploys individual CronJobs to each node, allowing for node-specific periodic tasks while maintaining standard DaemonSet-like tolerations.

## Getting Started

### Prerequisites

- direnv 2.37.1
- go version v1.26.2+
- docker version 29.4.2+
- Access to a Kubernetes v1.35.3+ cluster.

### Deploy to the cluster

#### [Using the installer](manifests/install.yaml)

```sh
kubectl apply -n daemonjob-system --server-side=true -f https://raw.githubusercontent.com/github.com/berquerant/daemonjob/<tag or branch>/manifests/install.yaml
```

#### [Using Helm](charts/daemonjob)

``` sh
helm install -n daemonjob-system --create-namespace oci://ghcr.io/berquerant/daemonjob/charts/daemonjob/daemonjob
```

### Documentation

- [CRDs](docs)
- [Samples](config/samples)

## Development

First of all, you need to run `make init` to initialize the repository.

- `make` to genarate codes and build the manager binary.
- `make docker-build` to build the manager and broadcast images.
- `make test` to run unit tests.
- `make test-e2e` to run E2E tests.
- `make local` to create a local cluster, deploy the manager and [samples](config/samples).

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

### Release

1. Confirm that CI on the main branch is green.
2. Change [VERSION](VERSION) and commit it.
3. `make release`.
