# Git tag describe.
TAG = $(shell git describe --tags --always --dirty)

# Current version of the project.
VERSION ?= $(TAG)

registry ?= docker.io

build:
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build  -mod vendor -v -o ./bin/app ./cmd/main.go

container:
	@docker build -f ./Dockerfile -t $(registry)/earthquake-alert:$(VERSION) .