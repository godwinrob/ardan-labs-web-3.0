SHELL := /bin/bash

run:
	go run app/services/sales-api/main.go | go run app/tooling/logfmt/main.go

# docker build

VERSION := 1.0
BUILD_REF := "local"

all: sales-api

sales-api:
	docker build \
		-f zarf/docker/dockerfile.sales-api \
		-t sales-api-arm64:$(VERSION) \
		--build-arg BUILD_REF=$(BUILD_REF) \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
		.

# k8s/kind

KIND_CLUSTER := rob-api-cluster

kind-up:
	kind create cluster \
		--image kindest/node:v1.21.1 \
		--name $(KIND_CLUSTER) \
		--config zarf/k8s/kind/kind-config.yaml
	kubectl config set-context --current --namespace=sales-system

kind-down:
	kind delete cluster --name $(KIND_CLUSTER)

kind-status:
	kubectl get nodes -o wide
	kubectl get svc -o wide
	kubectl get pods -o wide --watch --all-namespaces

kind-load:
	cd zarf/k8s/kind/sales-pod; kustomize edit set image sales-api-image=sales-api-arm64:$(VERSION)
	kind load docker-image sales-api-arm64:$(VERSION) --name $(KIND_CLUSTER)

kind-apply:
	kustomize build zarf/k8s/kind/sales-pod | kubectl apply -f -

kind-logs:
	kubectl logs -l app=sales --all-containers=true -f --tail=100 --namespace=sales-system | go run app/tooling/logfmt/main.go

kind-restart:
	kubectl rollout restart deployment sales-pod

kind-update: all kind-load kind-restart

kind-update-apply: all kind-load kind-apply

kind-describe:
	kubectl describe pod sales-pod --namespace=sales-system
