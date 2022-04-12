TAG=$(shell git rev-parse HEAD)
IMAGE=internal-registry.example.com/alertmanager-to-github:$(TAG)

DOCKER_BUILD ?= DOCKER_BUILDKIT=1 docker build --progress=plain

.PHONY: build
build:
	$(DOCKER_BUILD) --target export --output bin/ .

.PHONY: test
test:
	$(DOCKER_BUILD) --target unit-test .

.PHONY: clean
clean:
	rm -rf bin

.PHONY: docker-build
docker-build:
	$(DOCKER_BUILD) -t "$(IMAGE)" .

.PHONY: docker-push
docker-push: docker-build
	docker push "$(IMAGE)"
