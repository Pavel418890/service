SHELL := /bin/bash
TOKEN = `curl --user "admin@example.com:gophers" \
			http://localhost:3000/users/token/920ee610-06ee-4f4e-a105-8fb95be31155 | \
		sed -E 's/.*"Token":"?([^,"]*)"?.*/\1/'`

#==============================================================================
# Building containers
sales-api:
	docker build \
		-f zarf/docker/dockerfile.sales-api \
		-t sales-api-amd64:1.0 \
		--build-arg VCS_REF=`git rev-parse HEAD` \
		--build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
		.

#==============================================================================
# Running from within k8s/dev
kind-up:
	kind create cluster \
		--image kindest/node:v1.25.3 \
		--name plots-starter-cluster \
		--config zarf/k8s/dev/kind-config.yaml

kind-down:
	kind delete cluster --name plots-starter-cluster

kind-load:
	kind load docker-image sales-api-amd64:1.0 --name plots-starter-cluster

kind-services:
	kubectl kustomize ./zarf/k8s/dev | kubectl apply -f -

kind-status:
	kubectl get nodes
	kubectl get pods --watch

kind-status-full:
	kubectl describe pod -lapp=sales-api

kind-logs:
	kubectl logs -f --pod-running-timeout=1h -lapp=sales-api --all-containers=true -f

kind-sales-api: sales-api
	kind load docker-image sales-api-amd64:1.0 --name plots-starter-cluster
	kubectl delete pods -lapp=sales-api
#==============================================================================
run:
	go run app/sales-api/main.go

tidy:
	go mod tidy
	go mod vendor

test:
	go test  ./... -count=1
	staticcheck ./...

runa:
	go run app/admin/main.go

dashboard:
	expvarmon \
		-ports="0.0.0.0:4000" \
		-vars="build,requests,goroutines,errors,mem:memstats.Alloc"

load:
	hey \
		-m GET \
		-c 1000 \
		-n 10000000 \
		-H "Authorization: Bearer ${TOKEN}" \
		"http://localhost:3000/users/1/2"
