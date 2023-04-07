##############
# VARIABLES

GOLANG       := golang:1.20
ALPINE       := alpine:3.17
KIND         := kindest/node:v1.25.3
POSTGRES     := postgres:15-alpine
VAULT        := hashicorp/vault:1.12
ZIPKIN       := openzipkin/zipkin:2.23
TELEPRESENCE := datawire/tel2:2.12.1
TELE_MANAGER := datawire/ambassador-telepresence-manager:2.12.2
TELE_AGENT   := docker.io/curlimages/curl:latest
VERSION      := 1.0
ENVIRONMENT  := "development"
KIND_CLUSTER := rob-web-cluster

# http://sales-service.sales-system.svc.cluster.local:4000/debug/pprof/

run:
	go run app/services/sales-api/main.go | go run app/tooling/logfmt/main.go -service=SALES-API

tidy:
	go mod tidy
	go mod vendor


all: sales

sales:
	docker build \
		-f zcontain/docker/Dockerfile.sales-api \
		-t sales-api:$(VERSION) \
		--build-arg BUILD_REF=$(VERSION) \
		--build-arg ENVIRONMENT=$(ENVIRONMENT) \
		--build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
		.

kind-up:
	kind create cluster \
		--image kindest/node:v1.25.3 \
		--name $(KIND_CLUSTER) \
		--config zcontain/k8s/dev/kind-config.yaml
	kubectl wait --timeout=120s --namespace=local-path-storage --for=condition=Available deployment/local-path-provisioner

	kind load docker-image $(TELEPRESENCE) --name $(KIND_CLUSTER)
	kind load docker-image $(TELE_MANAGER) --name $(KIND_CLUSTER)
	kind load docker-image $(TELE_AGENT) --name $(KIND_CLUSTER)

	telepresence --context=kind-$(KIND_CLUSTER) helm install
	telepresence --context=kind-$(KIND_CLUSTER) connect

kind-down:
	telepresence quit -s
	kind delete cluster --name $(KIND_CLUSTER)

kind-load:
	kind load docker-image sales-api:$(VERSION) --name $(KIND_CLUSTER)

kind-apply:
	kustomize build zcontain/k8s/dev/sales | kubectl apply -f -
	kubectl wait --timeout=120s --namespace=sales-system --for=condition=Available deployment/sales

kind-restart:
	kubectl rollout restart deployment sales --namespace=sales-system

kind-status:
	kubectl get nodes -o wide
	kubectl get svc -o wide
	kubectl get pods -o wide --watch --all-namespaces

kind-describe:
	kubectl describe nodes
	kubectl describe svc

kind-describe-sales:
	kubectl describe pod --namespace=sales-system -l app=sales

kind-describe-traffic:
	kubectl describe pod --namespace=ambassador -l app=traffic-manager

kind-describe-uninstall:
	kubectl describe pod --namespace=ambassador -l app=uinstall-agents

kind-logs:
	kubectl logs --namespace=sales-system -l app=sales --all-containers=true -f --tail=100 --max-log-requests=6 | go run app/tooling/logfmt/main.go -service=SALES-API

metrics-local:
	expvarmon -ports=":4000" -vars="build,requests,goroutines,errors,panics,mem:memstats.Alloc"

kind-update: all kind-load kind-restart

kind-update-apply: all kind-load kind-apply
