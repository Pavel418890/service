SHELL := /bin/bash

sales-api:
	docker build \
		-f zarf/docker/dockerfile.sales-api \
		-t sales-api-amd64:1.0 \
		--build-arg VCS_REF=`git rev-parse HEAD` \
		--build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
		.
run:
	go run app/sales-api/main.go

tidy:
	go mod tidy
	go mod vendor

test:
	go test -v ./... -count=1
	staticcheck ./...

runa:
	go run app/admin/main.go

dashboard:
	expvarmon -ports=":4000" -vars="build,requests,goroutines,errors,mem:memstats.Alloc"

load:
	hey -m GET -c 100 -n 10000000 "http://localhost:3000/readiness"
