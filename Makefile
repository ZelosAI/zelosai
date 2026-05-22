# zelosai operator Makefile.
# Suite-standard targets: build, run, test, lint, image, push.

IMG ?= ghcr.io/zelosai/zelosai:develop
VERSION ?= $(shell cat VERSION 2>/dev/null || echo dev)
CONTROLLER_GEN ?= $(GOPATH)/bin/controller-gen
KUSTOMIZE ?= $(GOPATH)/bin/kustomize
GOPATH ?= $(HOME)/go

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
