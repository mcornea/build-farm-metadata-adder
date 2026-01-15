IMAGE ?= metadata-adder:latest

.PHONY: build
build:
	go build -o bin/metadata-adder .

.PHONY: container-build
container-build:
	podman build -t $(IMAGE) .

.PHONY: container-build-multiarch
container-build-multiarch:
	podman build --platform linux/amd64,linux/arm64 --manifest $(IMAGE) .

.PHONY: container-push
container-push:
	podman push $(IMAGE)

.PHONY: container-push-multiarch
container-push-multiarch:
	podman manifest push $(IMAGE) docker://$(IMAGE)

.PHONY: clean
clean:
	rm -rf bin/

.PHONY: test
test:
	go test ./...

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: all
all: fmt vet test build
