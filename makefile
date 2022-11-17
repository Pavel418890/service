SHELL := /bin/bash

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
