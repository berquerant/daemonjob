# daemonjob Project Analysis & AI Development Guidelines

This document provides a comprehensive overview of the **daemonjob** Kubernetes Operator project—its purpose, features, code structure, and development guidelines for AI assistants contributing to or maintaining this codebase.

---

## 1. Purpose & Overview

### Background & Objective
Standard Kubernetes **DaemonSets** are designed to run long-running daemon processes on every node, while **Jobs** are designed for run-to-completion batch workloads.
However, executing a batch workload **once per node (or targeted set of nodes) across the entire cluster** (e.g., node diagnostics, log collection, security scanning, cache warming) was difficult to achieve using standard resources alone.

**daemonjob** is a Kubernetes Operator that combines the node-targeting capabilities of a DaemonSet with the run-to-completion semantics of a Job through Custom Resource Definitions (CRDs) and a dedicated Controller.

---

## 2. Key Features & Custom Resources (CRDs)

The project introduces three main Custom Resource Definitions:

| CRD Name | API Path | Description & Behavior |
| :--- | :--- | :--- |
| **`DaemonJob`** | [api/v1/daemonjob_types.go](api/v1/daemonjob_types.go) | Executes a run-to-completion task once per node. The controller launches a middleman `broadcast Job`, which provisions and dispatches individual Worker Jobs to each targeted node. |
| **`DaemonCronJob`** | [api/v1/daemoncronjob_types.go](api/v1/daemoncronjob_types.go) | Extends `DaemonJob` with cron scheduling, triggering cluster-wide node-local executions periodically via a CronJob controller. |
| **`DaemonCronJobSet`** | [api/v1/daemoncronjobset_types.go](api/v1/daemoncronjobset_types.go) | A high-granularity scheduler that deploys individual CronJobs directly to each node, allowing node-specific scheduled tasks while maintaining standard DaemonSet-like tolerations. |

### Architectural Highlight: The Broadcast Pattern
- Rather than having the main controller (`bin/manager`) directly dispatch hundreds of Worker Jobs to all nodes, the controller launches a **Broadcast Container ([broadcast/broadcast.sh](broadcast/broadcast.sh))** inside a `broadcast Job`.
- The `broadcast.sh` script inspects the cluster's nodes and spawns the node-specific Worker Jobs, ensuring scalable and distributed job creation.

---

## 3. Directory Structure & Key Files

```text
daemonjob/
├── api/v1/                 # Go struct definitions for CRDs (DaemonJob, DaemonCronJob, DaemonCronJobSet)
├── broadcast/              # Broadcast container logic (broadcast.sh) and Dockerfile
├── cmd/main.go             # Controller Manager entrypoint
├── internal/
│   ├── controller/         # Reconciliation logic & controller unit tests
│   └── util/               # Internal utility packages
├── config/                 # Kustomize manifests (crd, default, manager, rbac, samples)
├── manifests/              # Consolidated installation manifest (install.yaml)
├── charts/daemonjob/       # Helm chart definitions
├── docs/                   # Auto-generated CRD documentation (daemonjob.berquerant.github.io.md)
├── hack/                   # Build, test, tool setup, and code generation scripts
├── Makefile                # Primary development tasks (build, test, generate, lint, etc.)
├── PROJECT                 # Kubebuilder (v4) project metadata
└── README.md               # Repository documentation and quickstart
```

---

## 4. Development & Verification Workflow

### Prerequisites
- Go `v1.26.2+`
- Docker `29.4.2+`
- direnv `2.37.1` (Recommended)
- Access to a Kubernetes cluster `v1.35.3+` (e.g., K0s, Kind)

### Essential Make Commands

```bash
# Repository Initialization
make init

# Code & Manifest Generation
make generate    # Generates DeepCopy method implementations
make manifests   # Generates CRD / RBAC manifests
make docs        # Generates CRD documentation via hack/docs.sh

# Linting & Code Quality
make fmt         # Runs go fmt
make vet         # Runs go vet
make lint        # Runs linters (go-fix, golangci-lint, check-licenses, shellcheck)

# Testing
make test        # Runs unit tests (using envtest)
make test-e2e    # Spawns a local K0s cluster and executes E2E tests

# Building & Local Deployment
make build       # Builds the manager binary (bin/manager)
make docker-build # Builds manager & broadcast container images
make local       # Creates a local cluster, loads images, deploys controller and samples

# Installer & Helm Chart Updates
make build-installer # Updates manifests/install.yaml
make chart           # Updates Helm chart in charts/daemonjob/
```

---

## 5. AI Assistant Development Guidelines

When modifying, extending, or refactoring this repository, AI assistants must adhere to the following principles:

### 1. Code Generation & Synchronization Rules
- **CRD Struct Changes**: When modifying structs in [api/v1/](api/v1/), immediately run `make generate` and `make manifests` to update [zz_generated.deepcopy.go](api/v1/zz_generated.deepcopy.go) and files in [config/crd/](config/crd/).
- **Deployment Asset & Doc Sync**: After updating resource specifications, run `make build-installer`, `make chart`, and `make docs` to keep [manifests/install.yaml](manifests/install.yaml), [charts/daemonjob/](charts/daemonjob/), and [docs/](docs/) synchronized.
- **License Headers**: Ensure all new Go files include the boilerplate license header defined in [hack/boilerplate.go.txt](hack/boilerplate.go.txt).

### 2. Strict Verification Protocols
- **Always Verify**: Never complete a task after making code edits without running verification commands. Run `make lint` and `make test` to verify your changes.
- **No Superficial Fixes**: Do not bypass failing tests by deleting assertions, disabling linters, or swallowing errors. Investigate full error tracebacks and fix root causes.

### 3. Respecting Architecture & Design Patterns
- **Reconciler Pattern**: Add new controller logic inside [internal/controller/](internal/controller/) following Kubebuilder Reconciler conventions. Always write or update corresponding unit tests (`*_test.go`).
- **Broadcast Mechanism**: Respect the `broadcast Job` -> `Worker Job` propagation flow (e.g., NodeSelector, Tolerations, OwnerReferences).

### 4. Definition of Done Checklist
1. [ ] Ran `make generate` and `make manifests`?
2. [ ] Updated manifests, charts, and docs via `make docs`, `make build-installer`, and `make chart`?
3. [ ] Passed `make lint` with zero errors?
4. [ ] Passed `make test` cleanly?
5. [ ] Verified with `make test-e2e` or `make local` when modifying controller behaviors?
