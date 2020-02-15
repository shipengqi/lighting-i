SHELL := /bin/bash
BASEDIR := $(shell pwd)
BINARY_NAME := lighting

.PHONY: build lint golint govet clean help

.ONESHELL:
build: lint govet
	@echo "build binary: $(BINARY_NAME) ..."
	@CGO_ENABLED=0 go build -o $(BINARY_NAME) ./main.go
	@echo "build $(BINARY_NAME) completed."

lint:
	@echo "run go lint ..."
	@golint $(BASEDIR)

golint:
	@echo "install golint ..."
	@go get -u golang.org/x/lint/golint
	@go install golang.org/x/lint/golint

.ONESHELL:
govet:
	@echo "run go vet ..."
	@go vet ./main.go

clean:
	@echo "clean ..."
	@rm -rf $(BINARY_NAME)

.PHONY: show
show:
	@echo "current directory: $(BASEDIR)"

help:
	@echo "make              - compile the source code"
	@echo "make golint       - install golint"
	@echo "make lint         - run go lint"
	@echo "make vet          - run go vet"
	@echo "make clean        - remove binary file"
