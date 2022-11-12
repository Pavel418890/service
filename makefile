SHELL := /bin/bash

run:
	go run app/sales-api/main.go

tidy:
	go mod tidy
	go mod vendor

build:
	go build app/sales-api/main.go
