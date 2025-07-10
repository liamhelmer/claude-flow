# Claude Flow Swarm Operator Makefile

# Image URL to use all building/pushing image targets
REGISTRY ?= ghcr.io
ORG ?= claude-flow
IMAGE_NAME ?= swarm-operator
VERSION ?= latest
IMG ?= $(REGISTRY)/$(ORG)/$(IMAGE_NAME):$(VERSION)
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.29.0

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: dev-up
dev-up: ## Start development environment
	docker-compose up -d dev
	@echo "Development container started. Run 'make dev-shell' to enter."

.PHONY: dev-down
dev-down: ## Stop development environment
	docker-compose down

.PHONY: dev-shell
dev-shell: ## Enter development container shell
	docker-compose exec dev /bin/bash

.PHONY: dev-build
dev-build: ## Build development container
	docker-compose build dev

.PHONY: init
init: ## Initialize kubebuilder project (run inside dev container)
	kubebuilder init --domain claudeflow.io --repo github.com/claude-flow/swarm-operator

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile cover.out

.PHONY: test-unit
test-unit: manifests generate fmt vet ## Run unit tests only.
	go test ./pkg/... ./internal/... -coverprofile cover.out

.PHONY: test-integration
test-integration: manifests generate fmt vet envtest ## Run integration tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./controllers/... ./internal/controller/... -coverprofile cover.out

.PHONY: test-e2e
test-e2e: manifests generate fmt vet ## Run E2E tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./e2e/... -timeout 30m -v

.PHONY: test-coverage
test-coverage: test ## Generate test coverage report.
	go tool cover -html=cover.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: test-benchmark
test-benchmark: ## Run benchmark tests.
	go test -bench=. -benchmem ./...

.PHONY: test-race
test-race: manifests generate fmt vet envtest ## Run tests with race detector.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test -race ./...

.PHONY: test-ci
test-ci: manifests generate fmt vet lint test-race ## Run all tests for CI.
	@echo "All CI tests passed!"

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	./scripts/build-image.sh --version $(VERSION)

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	./scripts/build-image.sh --version $(VERSION) --push

.PHONY: docker-build-multiarch
docker-build-multiarch: ## Build multi-architecture docker image.
	./scripts/build-multiarch.sh

.PHONY: docker-scan
docker-scan: ## Scan docker image for vulnerabilities.
	./scripts/test-image.sh $(IMG)

.PHONY: docker-sign
docker-sign: ## Sign docker image with cosign.
	cosign sign --yes $(IMG)

##@ Deployment

.PHONY: install
install: manifests ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	kustomize build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	kustomize build config/crd | kubectl delete --ignore-not-found=true -f -

.PHONY: deploy
deploy: manifests ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	kustomize build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	kustomize build config/default | kubectl delete --ignore-not-found=true -f -

##@ Testing

.PHONY: kind-create
kind-create: ## Create a kind cluster for testing
	kind create cluster --name swarm-operator-test

.PHONY: kind-delete
kind-delete: ## Delete the kind cluster
	kind delete cluster --name swarm-operator-test

.PHONY: kind-load
kind-load: docker-build ## Load operator image into kind cluster
	kind load docker-image ${IMG} --name swarm-operator-test

##@ Documentation

.PHONY: docs
docs: ## Generate documentation
	@mkdir -p docs
	@echo "Documentation generation not yet implemented"

##@ Security

.PHONY: security-scan
security-scan: ## Run gosec security scanner
	@echo "Running security scan..."
	gosec -fmt json -out security-report.json ./... || true
	@echo "Security scan complete. Report saved to security-report.json"

.PHONY: sbom
sbom: ## Generate Software Bill of Materials
	@echo "Generating SBOM..."
	syft packages dir:. -o spdx-json > sbom.spdx.json
	@echo "SBOM saved to sbom.spdx.json"

.PHONY: security-policy
security-policy: ## Apply security policies to cluster
	kubectl apply -f config/security/

##@ Helm

HELM_CHART_DIR = helm/swarm-operator
HELM_CHART_VERSION = $(shell grep 'version:' $(HELM_CHART_DIR)/Chart.yaml | awk '{print $$2}')

.PHONY: helm-lint
helm-lint: ## Lint the Helm chart
	@echo "Linting Helm chart..."
	helm lint $(HELM_CHART_DIR)

.PHONY: helm-package
helm-package: helm-lint ## Package the Helm chart
	@echo "Packaging Helm chart version $(HELM_CHART_VERSION)..."
	helm package $(HELM_CHART_DIR) -d dist/

.PHONY: helm-install
helm-install: ## Install the Helm chart locally
	helm install swarm-operator $(HELM_CHART_DIR) \
		--namespace swarm-system \
		--create-namespace \
		--set image.tag=$(VERSION)

.PHONY: helm-upgrade
helm-upgrade: ## Upgrade the Helm chart installation
	helm upgrade swarm-operator $(HELM_CHART_DIR) \
		--namespace swarm-system \
		--set image.tag=$(VERSION)

.PHONY: helm-uninstall
helm-uninstall: ## Uninstall the Helm chart
	helm uninstall swarm-operator --namespace swarm-system

.PHONY: helm-test
helm-test: ## Run Helm chart tests
	helm test swarm-operator --namespace swarm-system

.PHONY: helm-docs
helm-docs: ## Generate Helm documentation
	@echo "Generating Helm documentation..."
	@if command -v helm-docs &> /dev/null; then \
		helm-docs --chart-search-root $(HELM_CHART_DIR); \
	else \
		echo "helm-docs not installed. Install with: brew install helm-docs"; \
	fi

.PHONY: helm-template
helm-template: ## Render Helm chart templates locally
	helm template swarm-operator $(HELM_CHART_DIR) \
		--namespace swarm-system \
		--set image.tag=$(VERSION)

##@ Build Dependencies

# LOCALBIN is where project binaries are installed
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUBECTL ?= kubectl
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
KUSTOMIZE ?= $(LOCALBIN)/kustomize
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint

## Tool Versions
CONTROLLER_TOOLS_VERSION ?= v0.14.0
KUSTOMIZE_VERSION ?= v5.3.0
GOLANGCI_LINT_VERSION ?= v1.54.2

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || GOBIN=$(LOCALBIN) go install sigs.k8s.io/kustomize/kustomize/v5@$(KUSTOMIZE_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	test -s $(LOCALBIN)/golangci-lint || GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

##@ Utilities

.PHONY: clean
clean: ## Clean build artifacts
	rm -rf bin/
	rm -rf cover.out coverage.html
	rm -rf vendor/
	rm -f security-report.json sbom.spdx.json

.PHONY: verify
verify: ## Verify all generated code is up to date
	@echo "Verifying generated code..."
	@make manifests generate
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Generated code is out of date. Please run 'make manifests generate'"; \
		exit 1; \
	fi

.PHONY: lint
lint: golangci-lint ## Run golangci-lint
	$(GOLANGCI_LINT) run --timeout=5m

.PHONY: deps
deps: ## Download dependencies
	go mod download
	go mod tidy