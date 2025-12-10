REGISTRY ?= docker.io
IMAGE_NAME ?= mikedillon89/guest-lock-manager
TAG ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo latest)
IMAGE := $(REGISTRY)/$(IMAGE_NAME):$(TAG)
PLATFORMS ?= linux/amd64,linux/arm64
DATA_DIR ?= $(PWD)/data
APP_PORT ?= 8099
APP_URL ?= http://localhost:$(APP_PORT)
DOCKER_BUILDKIT ?= 1
VERSION_FROM_CONFIG := $(shell sed -n 's/^version:[[:space:]]*"\(.*\)"/\1/p' config.yaml | head -n 1)

.PHONY: docker-build docker-push docker-buildx-push docker-run api-health api-status release

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

release:
	@set -e; \
	ver="$(VERSION_FROM_CONFIG)"; \
	if [ -z "$$ver" ]; then echo "Could not read version from config.yaml"; exit 1; fi; \
	echo "Detected version from config.yaml: $$ver"; \
	if curl -fsSL "https://hub.docker.com/v2/repositories/$(IMAGE_NAME)/tags/$$ver" >/dev/null 2>&1; then \
		echo "Tag $$ver already exists on Docker Hub."; \
		printf "Create a new version? [y/N]: "; read ans; \
		if [ "$$ans" = "y" ] || [ "$$ans" = "Y" ]; then \
			printf "Enter new version: "; read newver; \
			if [ -z "$$newver" ]; then echo "No version provided, aborting."; exit 1; fi; \
			python - "$$newver" <<'PY' || exit 1
import sys, re, pathlib
ver=sys.argv[1]
path=pathlib.Path("config.yaml")
text=path.read_text()
new=re.sub(r'^version:\s*".*"$', f'version: "{ver}"', text, count=1, flags=re.MULTILINE)
path.write_text(new)
PY
			ver="$$newver"; \
		else \
			printf "Overwrite existing tag? [y/N]: "; read ow; \
			if [ "$$ow" != "y" ] && [ "$$ow" != "Y" ]; then echo "Aborting."; exit 1; fi; \
		fi; \
	fi; \
	echo "Building image $(REGISTRY)/$(IMAGE_NAME):$$ver with VERSION=$$ver"; \
	DOCKER_BUILDKIT=$(DOCKER_BUILDKIT) docker build --build-arg VERSION=$$ver -t $(REGISTRY)/$(IMAGE_NAME):$$ver .; \
	echo "Pushing tag $$ver..."; \
	docker push $(REGISTRY)/$(IMAGE_NAME):$$ver; \
	printf "Commit config.yaml with version $$ver and push to git? [y/N]: "; read commit; \
	if [ "$$commit" = "y" ] || [ "$$commit" = "Y" ]; then \
		git add config.yaml; \
		git commit -m "Bump version to $$ver"; \
		git push; \
	else \
		echo "Skipping git commit/push."; \
	fi


