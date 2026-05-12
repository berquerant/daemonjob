# Determine app version.
ifeq (true,$(DEV))
VERSION := 0.0.0
else
VERSION := $(shell ./hack/version.sh)
endif
# Image URL to use all building/pushing image targets.
IMAGE_TAG := $(VERSION)
IMAGE ?= $(IMAGE_REGISTRY)/manager:$(IMAGE_TAG)
BROADCAST_IMAGE ?= $(IMAGE_REGISTRY)/broadcast:$(IMAGE_TAG)
BROADCAST_IMAGE_REPOSITORY ?= $(IMAGE_REGISTRY)/broadcast
# YEAR defines the year value used for substituting the YEAR placeholder in the boilerplate header.
YEAR ?= $(shell date +%Y)
# THIRD_PARTY_LICENSES is where a report of licenses is generated.
THIRD_PARTY_LICENSES := NOTICE
# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

## Tool Binaries
TOOL ?= $(shell pwd)/hack/tool.sh
CLEAN ?= $(TOOL) clean
KUBECTL ?= $(TOOL) kubectl
KUSTOMIZE ?= $(TOOL) kustomize
CONTROLLER_GEN ?= $(TOOL) controller-gen
ENVTEST ?= $(TOOL) setup-envtest
GOLANGCI_LINT ?= $(TOOL) golangci-lint
HELMIFY ?= $(TOOL) helmify
CLUSTER ?= $(shell pwd)/hack/cluster.sh
INSTALLER ?= $(shell pwd)/hack/build-installer.sh

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: init
init: .env.github ## Setup the repository.
	mkdir -p $(CACHE_DIR)
	@echo 'Please run: direnv allow'

.PHONY: .env.github
.env.github: .github/actions/setup-env/action.yml ## Generate .env from setup-env action.
	yq '.runs.steps[0].run' $< | awk '/.+=.+/' > $@

##@ Development

.PHONY: manifests
manifests: ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt",year=$(YEAR) paths="./..."

.PHONY: $(THIRD_PARTY_LICENSES)
$(THIRD_PARTY_LICENSES): ## Generate NOTICE.
	./hack/license.sh report > $@

.PHONY: check-licenses-diff
check-licenses-diff: $(THIRD_PARTY_LICENSES) ## Fail if NOTICE has uncommitted changes.
	git diff --exit-code $<

.PHONY: check-licenses
check-licenses: check-licenses-diff ## Check whether licenses are not allowed.
	./hack/license.sh check

.PHONY: docs
docs: ## Generate CRD docs.
	./hack/docs.sh

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: vuln
vuln: ## Run govolncheck.
	go tool govulncheck ./...

.PHONY: go-fix
go-fix: ## Run go fix.
	go fix -diff ./...

.PHONY: go-fix-fix
go-fix-fix: ## Run go fix and perform fixes.
	go fix ./...

.PHONY: golangci-lint
golangci-lint: golangci-lint-config ## Run golangci-lint linter.
	$(GOLANGCI_LINT) run

.PHONY: golangci-lint-fix
golangci-lint-fix: golangci-lint-config ## Run golangci-lint linter and perform fixes.
	$(GOLANGCI_LINT) run --fix

.PHONY: golangci-lint-config
golangci-lint-config: ## Verify golangci-lint linter configuration.
	$(GOLANGCI_LINT) config verify

LINT_TASKS := vet go-fix golangci-lint check-licenses

.PHONY: lint
lint: ## Run linters.
	$(MAKE) -j $(words $(LINT_TASKS)) $(LINT_TASKS)

.PHONY: test
test: manifests generate fmt ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST))" go test $$(go list ./... | grep -v /e2e) -coverprofile cover.out

# TODO(user): To use a different vendor for e2e tests, modify the setup under 'tests/e2e'.
# The default setup assumes K0s is pre-installed and builds/loads the Manager Docker image locally.
# kubectl kuberc is disabled by default for test isolation; enable with:
# - KUBECTL_KUBERC=true
# CertManager is installed by default; skip with:
# - SKIP_CERT_MANAGER_INSTALL=true
# Recreate cluster by default; skip with
# - SKIP_DESTROY_CLUSTER=true
# Build images by default; skip with
# - SKIP_DOCKER_BUILD=true

.PHONY: setup-test-e2e
setup-test-e2e: recreate-cluster docker-build load ## Set up a cluster for e2e tests if it does not exist.

.PHONY: test-e2e
test-e2e: setup-test-e2e manifests generate fmt vet ## Run the e2e tests. Expected an isolated environment using K0s. CertManager is installed by default, skip with: SKIP_CERT_MANAGER_INSTALL=true Recreate cluster by default, skip with SKIP_DESTROY_CLUSTER=true Build images by default, skip with SKIP_DOCKER_BUILD=true
	go test -tags=e2e ./test/e2e/ -v -ginkgo.v
	$(MAKE) cleanup-test-e2e

.PHONY: cleanup-test-e2e
cleanup-test-e2e: destroy-cluster ## Tear down the cluster used for e2e tests.

.PHONY: clean ## Clean generated files.
clean: clean-binaries

.PHONY: clean-binaries
clean-binaries: ## Clean all binaries.
	$(CLEAN)

##@ Build

.PHONY: build
build: manifests generate docs fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

# If you wish to build the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: ## Build docker image with the manager.
ifeq (true,$(SKIP_DOCKER_BUILD))
	@echo Skip docker-build because SKIP_DOCKER_BUILD is true.
else

ifeq (true,$(DOCKER_CACHE_READONLY))
	IMAGE_TAG=$(IMAGE_TAG) $(CONTAINER_TOOL) buildx bake --set "*.cache-from=type=local,src=$(DOCKER_CACHE_DIR)" --allow=fs=$(CACHE_DIR) --load
else
	IMAGE_TAG=$(IMAGE_TAG) $(CONTAINER_TOOL) buildx bake --set "*.cache-from=type=local,src=$(DOCKER_CACHE_DIR)" --set "*.cache-to=type=local,mode=max,dest=$(DOCKER_CACHE_DIR)" --allow=fs=$(CACHE_DIR) --load
endif

endif

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	$(CONTAINER_TOOL) push ${IMAGE}

.PHONY: build-installer
build-installer: manifests generate ## Generate a consolidated YAML with CRDs and deployment.
	$(INSTALLER) manifest

.PHONY: chart
chart: build-installer ## Generate Helm charts. See https://github.com/arttor/helmify
	$(INSTALLER) chart $(VERSION)

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	@out="$$( $(KUSTOMIZE) build config/crd 2>/dev/null || true )"; \
	if [ -n "$$out" ]; then echo "$$out" | $(KUBECTL) apply -f - --server-side=true; else echo "No CRDs to install; skipping."; fi

.PHONY: uninstall
uninstall: manifests ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	@out="$$( $(KUSTOMIZE) build config/crd 2>/dev/null || true )"; \
	if [ -n "$$out" ]; then echo "$$out" | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -; else echo "No CRDs to delete; skipping."; fi

.PHONY: deploy
deploy: manifests ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	$(MAKE) deploy-kustomize-manager
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f - --server-side=true

.PHONY: deploy-kustomize-manager
deploy-kustomize-manager: ## Kustomize manager manifest to deploy.
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMAGE) && $(KUSTOMIZE) edit set configmap controller-manager --from-literal=BROADCAST_IMAGE_REPOSITORY=$(BROADCAST_IMAGE_REPOSITORY) --from-literal=BROADCAST_IMAGE_TAG=$(IMAGE_TAG)

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: create-cluster
create-cluster: ## Create a cluster.
	@(if $(CLUSTER) exist ; then echo "Cluster already exists. Skipping creation." ; else $(CLUSTER) start cluster ; fi)

.PHONY: destroy-cluster
destroy-cluster: ## Destroy a cluster.
ifeq (true,$(SKIP_DESTROY_CLUSTER))
	@echo Skip destroy-cluster because SKIP_DESTROY_CLUSTER is true.
else
	$(CLUSTER) stop cluster || true
endif

.PHONY: recreate-cluster
recreate-cluster: destroy-cluster create-cluster ## Recreate a cluster.

.PHONY: load
load: ## Load images to the cluster.
	$(CLUSTER) load $(IMAGE)
	$(CLUSTER) load $(BROADCAST_IMAGE)

.PHONY: samples
samples: ## Deploy config.samples to the cluster.
	$(KUBECTL) delete -k config/samples --ignore-not-found=true
	$(KUBECTL) apply -k config/samples

.PHONY: local
local: docker-build create-cluster load deploy restart-controller samples ## Build images, deploy controller and samples.

.PHONY: restart-controller
restart-controller: ## Restart controller deployment.
	kubectl -n daemonjob-system rollout restart deploy || true

.PHONY: release
release: ## Release images, manifest and chart.
	git tag "$(VERSION)"
	git push origin "$(VERSION)"
