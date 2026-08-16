# AGENTS.md - Antigravity Agent Guidelines for daemonjob

This file provides rules, architectural context, development workflows, and strict verification protocols for AI coding assistants working in the **daemonjob** repository.

---

## 1. Project Overview & Architecture

**daemonjob** is a Kubernetes Operator that combines the node-targeting capabilities of a DaemonSet with the run-to-completion batch semantics of a Job through Custom Resource Definitions (CRDs).

### Key Custom Resources (CRDs)
- **`DaemonJob`** (`api/v1/daemonjob_types.go`): Executes a batch job once per targeted node. Spawns a middleman broadcast Job.
- **`DaemonCronJob`** (`api/v1/daemoncronjob_types.go`): Adds cron-based scheduled executions for cluster-wide node batch jobs.
- **`DaemonCronJobSet`** (`api/v1/daemoncronjobset_types.go`): Deploys individual CronJobs directly to each targeted node.

### Architectural Pattern: Broadcast Execution
- The main controller manager (`cmd/controller/main.go`, `internal/controller/`) launches a `broadcast Job` running `cmd/broadcast/main.go`.
- The broadcast runner inspects nodes, constructs worker Job structs, validates them with `--dry-run=server`, and applies them to all targeted nodes atomically.

---

## 2. Directory Structure

```text
daemonjob/
├── api/v1/                 # CRD Go struct definitions (DaemonJob, DaemonCronJob, DaemonCronJobSet)
├── broadcast/              # Broadcast container Dockerfile
├── cmd/
│   ├── broadcast/          # Broadcast CLI entrypoint (main.go)
│   ├── controller/         # Controller Manager entrypoint (main.go)
│   └── daemonjobctl/       # daemonjobctl CLI entrypoint (main.go)
├── internal/
│   ├── broadcast/          # Broadcast runner logic & unit tests
│   ├── controller/         # Reconciler logic & unit tests
│   ├── daemonjobctl/       # daemonjobctl logic, golden tests, testdata/
│   └── util/               # Internal utilities
├── config/                 # Kustomize manifests (crd, default, manager, rbac, samples)
├── manifests/              # Consolidated installation manifest (install.yaml)
├── charts/daemonjob/       # Helm chart definitions
├── docs/                   # Auto-generated CRD documentation
├── hack/                   # Build, test, tool setup, and code generation scripts
├── Makefile                # Primary development tasks
└── PROJECT                 # Kubebuilder (v4) project metadata
```

---

## 3. Essential Development & Code Sync Workflows

### Code Generation & Manifest Synchronization
When modifying CRD structs, specifications, or manifests, you MUST run the corresponding generation commands:

1. **CRD Struct Changes (`api/v1/*.go`)**:
   - Run `make generate` (updates `zz_generated.deepcopy.go`)
   - Run `make manifests` (updates `config/crd/bases/`)
2. **Release Asset & Doc Sync**:
   - Run `make build-installer` (updates `manifests/install.yaml`)
   - Run `make chart` (updates `charts/daemonjob/`)
   - Run `make docs` (updates `docs/`)
3. **Golden Test Files**:
   - When modifying `daemonjobctl` output specifications or templates, update golden files via `make golden`.

---

## 4. Verification Protocols (Definition of Done)

Before declaring any task or PR complete, you MUST execute and pass all verification steps:

```bash
# 1. Code Quality & Linting
make fmt
make vet
make lint        # Runs go-fix, golangci-lint, check-licenses, shellcheck

# 2. Unit Tests
make test        # Runs controller & internal unit tests with envtest

# 3. End-to-End Tests (Mandatory for controller/broadcast changes)
make test-e2e    # Spawns local K0s cluster and executes full E2E suite
```

### Strict Quality Rules
- **No Swallowing Errors**: Never delete test assertions or ignore failing linters to pass CI. Investigate root causes and fix them cleanly.
- **License Headers**: All new Go files must include the license header specified in `hack/boilerplate.go.txt`.
- **ShellCheck**: All shell script changes (`*.sh`) must pass `shellcheck` with zero warnings.

---

## 5. Kubernetes Dependency Upgrade Playbook

When asked to upgrade Kubernetes libraries (e.g. `k8s.io/*`, `sigs.k8s.io/controller-runtime`), AI agents MUST follow this playbook:

### Step 1: Version Selection Matrix
1. **Target Kubernetes Version**: Choose `v1.Y.Z` (e.g. `v1.31.2`).
2. **`k8s.io/*` Submodules**: Must match `v0.Y.Z` (e.g. `v0.31.2`).
   - Dynamically inspect `go.mod` (e.g. `go list -m all | grep -E '^k8s\.io/'`) at execution time to identify active `k8s.io/*` dependencies rather than relying on hardcoded module lists.
3. **`controller-runtime` (`CR`)**: Select matching version from [controller-runtime Releases](https://github.com/kubernetes-sigs/controller-runtime/releases).
4. **`controller-tools` (`CT`)**: Select matching version from [controller-tools Releases](https://github.com/kubernetes-sigs/controller-tools/releases).
5. **`k0s` (`K0S`)**: Select closest matching version for E2E testing from [k0s Releases](https://github.com/k0sproject/k0s/releases) (e.g. `v1.31.2-k0s.0`).

### Step 2: Execute Update Helper Script
Run `./hack/update-k8s-deps.sh` to update `go.mod` and `.github/actions/setup-env/action.yml`:

```bash
# Usage: ./hack/update-k8s-deps.sh <k8s-version> <controller-runtime-version> [controller-tools-version] [k0s-version]
./hack/update-k8s-deps.sh 1.31.2 0.19.3 v0.16.5 v1.31.2-k0s.0
```

### Step 3: Full Manifest Re-generation & DoD Verification
After updating dependencies, execute the mandatory code generation, asset sync, and verification pipeline:

```bash
# 1. Code & Manifest Re-generation
make generate        # Updates zz_generated.deepcopy.go
make manifests       # Updates config/crd/bases/
make build-installer # Updates manifests/install.yaml
make chart           # Updates charts/daemonjob/
make docs            # Updates docs/

# 2. Quality & Verification
make fmt
make vet
make lint
make test
make test-e2e        # Must pass all E2E specs
```
