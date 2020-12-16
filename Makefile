SHELL := bash

server := cmd/doorman/doorman
cli := cmd/doormanc/doormanc

binaries := ${server} ${cli}
servers := ${server} ${server}-x86_64-linux
clis := ${cli} ${cli}-x86_64-linux
all: ${servers} ${clis}

clis := ${cli} $(addprefix ${cli}-x86_64-,darwin freebsd linux windows.exe)

.PHONY: ${servers} ${clis} test releases e2e
%-darwin: GOOS=darwin
%-freebsd: GOOS=freebsd
%-linux: GOOS=linux
%-windows.exe: GOOS=windows
${servers} ${clis}: protobuf/vpn_service.pb.go
${servers} ${clis}:
	CGO_ENABLED=0 GOOS=${GOOS} go build -o $@ ./$(@D)

MKDOCSIMAGE = docker.io/squidfunk/mkdocs-material:5.2.3

ifeq ($(origin GOBIN), undefined)
$(warning Warning: GOBIN/PATH handling in Makefile is deprecated and will be remove when direnv/.envrc use is made a requirement for Equinix projects)
export GOBIN := ${PWD}/bin
export PATH := ${GOBIN}:${PATH}
endif

bin/protoc-gen-go: #$(shell git ls-files 'vendor/github.com/golang/protobuf')
	go install github.com/golang/protobuf/protoc-gen-go

bin/cobra: #$(shell git ls-files 'vendor/github.com/spf13')
	go install github.com/spf13/cobra/cobra

protobuf/vpn_service.pb.go: bin/protoc-gen-go protobuf/vpn_service.proto
	protoc --go_out=plugins=grpc:./protobuf/ -I=./protobuf/ ./protobuf/*.proto

help: ## Print this help
	@(export SEP=$$'\01'; grep -E "^[a-zA-Z0-9_-]+:.*?## .*$$" $(MAKEFILE_LIST) | sed "s/:.*##/$$SEP/" | column -ts "$$SEP" -c 120)

test: ## Run go test for this project
	go test -race -coverprofile=coverage.txt -covermode=atomic ${TEST_ARGS} ./...

releases: $(filter-out ${cli},${clis}) ## Build all the releases

e2e: ${server]-x86_64-linux ${cli}-x86_64-linux ## Test openvpn connection in Vagrant
	mkdir -p vagrant-sync/
	vagrant up
	#vagrant destroy -f

run: ${server}-x86_64-linux ${cli}-x86_64-linux ## Run with docker compose
	docker-compose up --build server

documentation: ## Generate the documentation site under $OUTPUT_DIR (default: ./docs)
	@docker run --rm -it -p 8000:8000 -v ${PWD}:/docs ${MKDOCSIMAGE}
