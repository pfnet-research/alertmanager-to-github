TAG := $(shell git describe --tags --always --dirty)
IMAGE ?= ghcr.io/pfnet-research/alertmanager-to-github:$(TAG)
ARCH ?= amd64
ALL_ARCH ?= amd64 arm64

DOCKER_BUILD ?= DOCKER_BUILDKIT=1 docker build --progress=plain

.PHONY: build
build:
	$(DOCKER_BUILD) --target export --output bin/ .

.PHONY: test
test:
	$(DOCKER_BUILD) --target unit-test .

.PHONY: lint
lint:
	$(DOCKER_BUILD) --target lint .

.PHONY: clean
clean:
	rm -rf bin

.PHONY: docker-build
docker-build:
	$(DOCKER_BUILD) --pull --progress=plain --platform $(ARCH) -t $(IMAGE)-$(ARCH) .

docker-build-%:
	$(MAKE) ARCH=$* docker-build

.PHONY: docker-build-all
docker-build-all: $(addprefix docker-build-,$(ALL_ARCH))

.PHONY: docker-push
docker-push:
	docker push $(IMAGE)-$(ARCH)

docker-push-%:
	$(MAKE) ARCH=$* docker-push

.PHONY: docker-push-all
docker-push-all: $(addprefix docker-push-,$(ALL_ARCH))

.PHONY: docker-manifest-push
docker-manifest-push:
	docker manifest create --amend $(IMAGE) $(addprefix $(IMAGE)-,$(ALL_ARCH))
	@for arch in $(ALL_ARCH); do docker manifest annotate --arch $${arch} $(IMAGE) $(IMAGE)-$${arch}; done
	docker manifest push --purge $(IMAGE)

.PHONY: push-all
push-all: docker-push-all docker-manifest-push
