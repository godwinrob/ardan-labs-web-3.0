SHELL := /bin/bash

run:
	go run main.go

# docker build

VERSION := 1.0

all: service

service:
	docker build \
		-f zarf/docker/dockerfile \
		-t service-arm64:$(VERSION) \
		--build-arg BUILD_REF=$(VERSION) \
		--build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
		.

# k8s/kind

KIND_CLUSTER := first-cluster

kind-up:
	kind create cluster \
		--image kindest/node:v1.21.1 \
		--name $(KIND_CLUSTER) \
		--config zarf/k8s/kind/kind-config.yaml
	kubectl config set-context --current --namespace=service-system

kind-down:
	kind delete cluster --name $(KIND_CLUSTER)

kind-status:
	kubectl get nodes -o wide
	kubectl get svc -o wide
	kubectl get pods -o wide --watch --all-namespaces

kind-load:
	kind load docker-image service-arm64:$(VERSION) --name $(KIND_CLUSTER)

kind-apply:
	cat zarf/k8s/base/service-pod/base-service.yaml | kubectl apply -f -

kind-logs:
	kubectl logs -l app=service --all-containers=true -f --tail=100 --namespace=service-system