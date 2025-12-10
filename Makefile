REGISTRY ?= docker.io
IMAGE_NAME ?= mikedillon89/guest-lock-manager
TAG ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo latest)
IMAGE := $(REGISTRY)/$(IMAGE_NAME):$(TAG)
PLATFORMS ?= linux/amd64,linux/arm64
DATA_DIR ?= $(PWD)/data
APP_PORT ?= 8099
APP_URL ?= http://localhost:$(APP_PORT)
DOCKER_BUILDKIT ?= 1

.PHONY: docker-build docker-push docker-buildx-push docker-run api-health api-status

docker-build:
	DOCKER_BUILDKIT=$(DOCKER_BUILDKIT) docker build -t $(IMAGE) .

docker-push: docker-build
	docker push $(IMAGE)

# Multi-arch build (amd64 + arm64) and push; requires buildx/qemu.
docker-buildx-push:
	DOCKER_BUILDKIT=$(DOCKER_BUILDKIT) docker buildx build --platform $(PLATFORMS) -t $(IMAGE) --push .

docker-run:
	mkdir -p $(DATA_DIR)
	docker run --rm -p $(APP_PORT):8099 -v $(DATA_DIR):/data $(IMAGE)

api-health:
	curl -s $(APP_URL)/api/health | jq .

api-status:
	curl -s $(APP_URL)/api/status | jq .

