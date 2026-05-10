REGISTRY ?= quay.io
REPOSITORY ?= $(REGISTRY)/eformat/gpu-config-plugin
IMG := $(REPOSITORY):latest
PODMAN_ARGS ?=

compile:
	yarn run build

podman-build:
	podman build $(PODMAN_ARGS) . -t ${IMG} -f Containerfile

podman-push: podman-build
	podman push ${IMG}

test-go:
	go test ./...

lint:
	yarn lint
