export APP_NAME := $(patsubst %,%,$(notdir $(shell pwd)))
DESC :=
PROJECT_URL := "https://github.com/gomatic/$(APP_NAME)"

SOURCES := $(wildcard *.go)

.PHONY : build linux darwin run container
.PHONY : help report
.DEFAULT_GOAL := help

PREFIX ?= usr/local

# Capture the
export RELEASE := $(shell lsb_release -rs 2>/dev/null)

# Capture the commit and branch
export BRANCH ?= $(shell git rev-parse --symbolic-full-name --abbrev-ref HEAD 2>/dev/null)
export COMMIT_ID := $(shell git log --pretty=format:'%h' -n 1 2>/dev/null)
export COMMIT_TIME := $(shell git show -s --format=%ct 2>/dev/null)
export VERSION := $(COMMIT_TIME)-$(COMMIT_ID)

export STARTD := $(shell pwd)
export THIS := $(abspath $(lastword $(MAKEFILE_LIST)))
export THISD := $(dir $(THIS))

build: $(APP_NAME) ## Make everything

$(APP_NAME) $(GOBIN)/$(APP_NAME): $(SOURCES)
	go vet
	go build -ldflags "-X $(PACKAGE)" -v -o $@

install: $(GOBIN)/$(APP_NAME) ## Install to GOBIN

#

linux: GOOS := linux
linux: GOARCH := amd64
linux: $(APP_NAME)-$(GOOS)-$(GOARCH) ## Compile Linux binary

darwin: GOOS := darwin
darwin: GOARCH := amd64
darwin: gateway-$(GOOS)-$(GOARCH) ## Compile Darwin binary

$(APP_NAME)-$(GOOS)-$(GOARCH): $(SOURCES)
	CGO_ENABLED=0 go build -ldflags="-s -w -X main.VERSION=$(VERSION)" -a -installsuffix cgo -o $@

container: ## Create Docker image from Linux binary.
	$(MAKE) linux GOOS=linux GOARCH=amd64
	docker build \
    --tag $(APP_NAME):latest .
	docker tag $(APP_NAME):latest $(APP_NAME)/$(BRANCH):latest
	docker tag $(APP_NAME)/$(BRANCH):latest $(APP_NAME)/$(BRANCH):$(VERSION)

up: DAEMON=
up down:
	docker-compose $@ $(DAEMON)


clean:
	rm -f $(APP_NAME)

help: ## This help.
	@echo $(APP_NAME)
	@echo $(PROJECT_URL)
	@echo Targets:
	@awk 'BEGIN {FS = ":.*?## "} / [#][#] / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)


report: ## Show variables
	@echo STARTD='"$(STARTD)"'
	@echo THIS='"$(THIS)"'
	@echo THISD='"$(THISD)"'
	@echo VERSION='"$(VERSION)"'
