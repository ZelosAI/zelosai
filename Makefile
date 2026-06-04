# zelosai operator Makefile.
# Suite-standard targets: build, run, test, lint, image, push.

IMG ?= ghcr.io/zelosai/zelosai:develop
VERSION ?= $(shell cat VERSION 2>/dev/null || echo dev)
CONTROLLER_GEN ?= $(GOPATH)/bin/controller-gen
KUSTOMIZE ?= $(GOPATH)/bin/kustomize
GOPATH ?= $(HOME)/go

# --- e2e (kind) smoke knobs --------------------------------------------------
KIND ?= kind
KUBECTL ?= kubectl
KIND_CLUSTER ?= zelosai-e2e
E2E_OPERATOR_IMG ?= zelos.local/zelosai:e2e
E2E_STUB_IMG ?= zelos.local/e2e-stub:latest
E2E_READY_TIMEOUT ?= 8m
E2E_GO_TIMEOUT ?= 12m

.PHONY: all
all: build

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: build
build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/manager ./cmd

.PHONY: run
run: manifests generate
	go run ./cmd

.PHONY: test
test:
	go test ./... -count=1

.PHONY: vet
vet:
	go vet ./...

.PHONY: lint
lint: vet
	gofmt -s -l . | tee /dev/stderr | (! read)

.PHONY: manifests
manifests:
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate:
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: install
install: manifests
	kubectl apply -f config/crd/bases

.PHONY: uninstall
uninstall:
	kubectl delete -f config/crd/bases --ignore-not-found

.PHONY: deploy
deploy: manifests
	$(KUSTOMIZE) build deploy/operator | kubectl apply -f -

.PHONY: undeploy
undeploy:
	$(KUSTOMIZE) build deploy/operator | kubectl delete --ignore-not-found -f -

.PHONY: image
image:
	docker build -t $(IMG) .

.PHONY: push
push:
	docker push $(IMG)

.PHONY: bundle
bundle: manifests
	$(KUSTOMIZE) build deploy/operator > deploy/operator/bundle.yaml

# --- e2e (kind) smoke --------------------------------------------------------
# NOTE: requires kind + docker + kubectl on the host. This is NOT runnable in
# the dev sandbox (no kind there); it runs in CI (.github/workflows/e2e.yml) or
# on a kind-capable host. The Go assertions live in test/e2e (build tag `e2e`).

.PHONY: e2e-images
e2e-images: ## Build the operator + e2e readiness-stub images locally.
	docker build -t $(E2E_OPERATOR_IMG) .
	docker build -t $(E2E_STUB_IMG) -f test/e2e/stub/Dockerfile .

.PHONY: kind-up
kind-up: ## Create the kind cluster (idempotent).
	$(KIND) get clusters | grep -qx $(KIND_CLUSTER) || $(KIND) create cluster --name $(KIND_CLUSTER) --wait 120s

.PHONY: kind-down
kind-down: ## Delete the kind cluster.
	$(KIND) delete cluster --name $(KIND_CLUSTER)

.PHONY: e2e-deploy
e2e-deploy: e2e-images kind-up ## Load images + install operator + e2e overlay into kind.
	$(KIND) load docker-image $(E2E_OPERATOR_IMG) --name $(KIND_CLUSTER)
	$(KIND) load docker-image $(E2E_STUB_IMG) --name $(KIND_CLUSTER)
	$(KUSTOMIZE) build deploy/operator | sed 's#ghcr.io/zelosai/zelosai:develop#$(E2E_OPERATOR_IMG)#g' | $(KUBECTL) apply -f -
	$(KUBECTL) -n zelos-system rollout status deploy/zelosai-controller-manager --timeout=180s
	$(KUSTOMIZE) build deploy/e2e | sed 's#zelos.local/e2e-stub:latest#$(E2E_STUB_IMG)#g' | $(KUBECTL) apply -f -

.PHONY: test-e2e
test-e2e: e2e-deploy ## Full kind smoke: bring up, install, assert ZelosPlatform Ready.
	E2E_READY_TIMEOUT=$(E2E_READY_TIMEOUT) go test -tags e2e ./test/e2e/... -count=1 -timeout $(E2E_GO_TIMEOUT) -v

.PHONY: vet-e2e
vet-e2e: ## Type-check the e2e harness without a cluster (sandbox-safe).
	go vet -tags e2e ./test/...
