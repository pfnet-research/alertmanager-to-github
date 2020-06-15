TAG=$(shell git rev-parse HEAD)
IMAGE=internal-registry.example.com/alertmanager-to-github:$(TAG)

.PHONY: build
build: statik
	go build -o bin/alertmanager-to-github .

.PHONY: docker-build
docker-build: statik
	docker build -t "$(IMAGE)" .

.PHONY: docker-push
docker-push: docker-build
	docker push "$(IMAGE)"

.PHONY: statik
statik:
	statik -src=templates -dest=pkg
