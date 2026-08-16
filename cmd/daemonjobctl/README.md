# daemonjobctl

`daemonjobctl` is a CLI helper tool for working with **daemonjob** Custom Resources (`DaemonJob`, `DaemonCronJob`, and `DaemonCronJobSet`).

## Features

- **`skeleton`**: Generates a minimal, ready-to-use skeleton YAML manifest for any daemonjob Custom Resource.
- **`generate`**: Simulates and previews the actual Kubernetes resources (ServiceAccount, ClusterRoleBinding, broadcast Jobs/CronJobs, and per-node worker Jobs) that would be created in the cluster given a Custom Resource manifest and a list of target node names.
  - Passes through non-daemonjob Kubernetes resources (such as `ConfigMap`, `Secret`, `Service`, etc.) unchanged.
  - Supports multi-document YAML streams (`---` separated).
  - Outlines simulated worker Jobs as commented-out YAML for easy inspection.

> **Note:** The output of `daemonjobctl generate` is for **preview and simulation purposes only** (e.g. CI review, static analysis, dry-run). It does not guarantee runtime cluster behavior, and runtime-assigned fields (such as controller UID) are placeholders.

## Installation

### Using `go install`

```sh
go install github.com/berquerant/daemonjob/cmd/daemonjobctl@latest
```

### Building from Source

In the root of the `daemonjob` repository:

```sh
make build
```

The binary will be generated at `./bin/daemonjobctl`.

Alternatively, build only `daemonjobctl`:

```sh
go build -o bin/daemonjobctl ./cmd/daemonjobctl/main.go
```

## Usage

### Command Summary

```text
Usage:
  daemonjobctl <command> [flags]

Commands:
  skeleton <kind>   Output a skeleton YAML for the given custom resource kind.
                    Supported kinds: daemonjob | daemoncronjob | daemoncronjobset

  generate          Generate the Kubernetes manifests that daemonjob would create
                    for a given custom resource and a list of node names.
                    This output is for preview and simulation purposes only (e.g. CI review, dry-run).
                    It does not guarantee runtime behavior, and runtime-assigned fields (such as
                    controller UID) are placeholders.
                    For DaemonJob and DaemonCronJob, broadcast resources are output
                    directly, and worker Jobs are output as commented-out YAML.
                    For DaemonCronJobSet, all per-node CronJobs are output directly.
```

---

### 1. Generating Skeletons (`skeleton`)

Generate skeleton Custom Resource definitions:

```sh
# Generate DaemonJob skeleton
daemonjobctl skeleton daemonjob > my-daemonjob.yaml

# Generate DaemonCronJob skeleton
daemonjobctl skeleton daemoncronjob > my-daemoncronjob.yaml

# Generate DaemonCronJobSet skeleton
daemonjobctl skeleton daemoncronjobset > my-daemoncronjobset.yaml
```

---

### 2. Simulating Deployed Manifests (`generate`)

Generate and preview the Kubernetes manifests that `daemonjob` controllers will create for a given node list.

#### Flags

| Flag | Description | Default |
|---|---|---|
| `-nodes` | Comma-separated list of node names (**required**) | `""` |
| `-f` | Input manifest YAML file path (reads from `stdin` if omitted) | `""` |
| `-broadcast-image` | Broadcast container image for `DaemonJob`/`DaemonCronJob` | `ghcr.io/berquerant/daemonjob-broadcast:latest` |
| `-broadcast-role` | ClusterRole name used by broadcast jobs | `daemonjob-broadcast-role` |

#### Examples

```sh
# Generate preview from a file
daemonjobctl generate -f config/samples/daemonjob_v1_daemonjob.yaml -nodes node-1,node-2,node-3

# Pipe from skeleton into generate
daemonjobctl skeleton daemonjob | daemonjobctl generate -nodes worker-1,worker-2

# Pipe through stdin
cat my-manifests.yaml | daemonjobctl generate -nodes node-a,node-b
```

#### Output Structure

- **`DaemonJob` / `DaemonCronJob`**:
  1. `ServiceAccount` and `ClusterRoleBinding` used by the broadcast runner.
  2. The broadcast `Job` or `CronJob` deployed by the operator.
  3. A commented-out `# Worker Jobs (simulated)` block showing the per-node worker `Job` definitions that the broadcast runner will create on each targeted node.
- **`DaemonCronJobSet`**:
  - Direct per-node `CronJob` definitions for each targeted node.
- **Other Resources (Pass-through)**:
  - Any standard Kubernetes resources (e.g. `ConfigMap`, `Secret`) in the input stream are passed through untouched.
