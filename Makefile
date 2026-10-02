GO ?= go
MODULE := github.com/barpilot/node-labeler-operator

.PHONY: build test generate
build:
	CGO_ENABLED=0 $(GO) build -trimpath -o bin/node-labeler-operator .

test:
	$(GO) test -race ./...

# Generators are pinned in go.mod; no Docker or GOPATH checkout is needed.
generate:
	$(GO) tool deepcopy-gen --output-file zz_generated.deepcopy.go ./apis/labeler/v1alpha1
	$(GO) tool client-gen --clientset-name versioned --input-base $(MODULE)/apis --input labeler/v1alpha1 --output-dir ./client/k8s/clientset --output-pkg $(MODULE)/client/k8s/clientset
