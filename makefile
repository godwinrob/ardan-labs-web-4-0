
run:
	go run app/services/sales-api/main.go

tidy:
	go mod tidy
	go mod vendor

VERSION := 1.0
ENVIRONMENT := "development"

all: sales

sales:
	docker build \
		-f zcontain/docker/Dockerfile.sales-api \
		-t sales-api:$(VERSION) \
		--build-arg BUILD_REF=$(VERSION) \
		--build-arg ENVIRONMENT=$(ENVIRONMENT) \
		--build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
		.

KIND_CLUSTER := rob-web-cluster

kind-up:
	kind create cluster \
		--image kindest/node:v1.25.3 \
		--name $(KIND_CLUSTER) \
		--config zcontain/k8s/dev/kind-config.yaml
	kubectl wait --timeout=120s --namespace=local-path-storage --for=condition=Available deployment/local-path-provisioner

kind-down:
	kind delete cluster --name $(KIND_CLUSTER)

kind-load:
	kind load docker-image sales-api:$(VERSION) --name $(KIND_CLUSTER)

kind-status:
	kubectl get nodes -o wide
	kubectl get svc -o wide
	kubectl get pods -o wide --watch --all-namespaces
