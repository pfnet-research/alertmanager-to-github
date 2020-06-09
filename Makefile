TAG=$(shell git rev-parse HEAD)
IMAGE=internal-registry.example.com/alertmanager-to-github:$(TAG)

.PHONY: docker-build
docker-build:
	docker build -t "$(IMAGE)" .

.PHONY: docker-push
docker-push: docker-build
	docker push "$(IMAGE)"
