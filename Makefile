#!make
.PHONY: build vendor gen

PROJECT := $(basename $(pwd))
VERSION := $(shell git describe --tags --always)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
export GOPRIVATE := github.com/dedovvlad/
export GOPROXY := direct

LOCAL_BIN := $(CURDIR)/bin
INTERNAL_PATH := $(CURDIR)/internal
RELATIVE_INTERNAL_PATH := ./internal

SWAGGER_BIN:=$(LOCAL_BIN)/swagger
SWAGGER_TAG=0.26.1

# swagger_darwin_amd64 mac m1, swagger_linux_arm64 for local build in docker; CI swagger_linux_amd64
SWAGGER_PLATFORM := $(if $(SWAGGER_PLATFORM),$(SWAGGER_PLATFORM),"swagger_darwin_amd64")

GOLANGCI_BIN:=$(LOCAL_BIN)/golangci-lint
GOLANGCI_TAG:=1.47.3

GOOSE_BIN:=$(LOCAL_BIN)/goose
GOOSE_TAG:=3.6.1
GOOSE_PLATFORM := $(if $(GOOSE_PLATFORM),$(GOOSE_PLATFORM),"darwin_arm64")

SQLC_BIN:=$(LOCAL_BIN)/sqlc
SQLC_TAG=1.14.0
SQLC_PLATFORM := $(if $(SQLC_PLATFORM),$(SQLC_PLATFORM),"darwin_amd64")

ifndef PROJECT
DIR = $(shell basename $(CURDIR))
PROJECT = $(subst _,-,${DIR})
endif

install-swagger:
ifeq ($(wildcard $(SWAGGER_BIN)),)
	@mkdir -p bin
	@echo "Downloading go-swagger $(SWAGGER_TAG)"
	@curl -o $(LOCAL_BIN)/swagger -L'#' "https://github.com/go-swagger/go-swagger/releases/download/v$(SWAGGER_TAG)/${SWAGGER_PLATFORM}"
	@chmod +x $(LOCAL_BIN)/swagger
endif

install-sqlc:
ifeq ($(wildcard $(SQLC_BIN)),)
	@mkdir -p bin
	@echo "Downloading sqlx ${SQLC_TAG} / ${SQLC_PLATFORM}"
	@wget -c "https://github.com/kyleconroy/sqlc/releases/download/v${SQLC_TAG}/sqlc_${SQLC_TAG}_${SQLC_PLATFORM}.tar.gz" -O - | tar -xz  -C ${LOCAL_BIN}
	@chmod +x ${LOCAL_BIN}/sqlc
endif
install-goose:
ifeq ($(wildcard $(GOOSE_BIN)),)
	@mkdir -p bin
	@echo "Downloading sqlx ${GOOSE_TAG} / ${GOOSE_PLATFORM}"
	@curl -o $(GOOSE_BIN) -L'#' "https://github.com/pressly/goose/releases/download/v${GOOSE_TAG}/goose_${GOOSE_PLATFORM}"
	@chmod +x ${GOOSE_BIN}
endif

install-lint:
ifeq ($(wildcard $(GOLANGCI_BIN)),)
	@echo "Downloading golangci-lint v$(GOLANGCI_TAG)"
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v$(GOLANGCI_TAG)
endif

## Dependencies
tidy:
	@go mod tidy

deps: tidy vendor

deps-client:
	@cd ./pkg/products-client;go mod tidy

## Generating files
gen: install-swagger clean-gen gen-clients gen-serv gen-serv-client deps

init-project: change-version mod-init

# Change version
change-version:
	ex -s +%s/template/${PROJECT}/ge -cwq version/version.go
	ex -s +%s/template/${PROJECT}/ge -cwq database/sqlc.yaml
	ex -s +%s/template/${PROJECT}/ge -cwq README.md

mod-init:
	@go mod init ${GOPRIVATE}${PROJECT}

# Генерация сервера
gen-serv:
	@mkdir -p cmd
	@mkdir -p internal/generated
	@mkdir -p internal/app
	@mkdir -p internal/config
	@mkdir -p internal/models
	@mkdir -p internal/processors && touch internal/processors/processor.go
	@mkdir -p internal/services/${PROJECT} && touch internal/services//${PROJECT}/service.go
	@${SWAGGER_BIN} generate server --allow-template-override \
		-f ./swagger-doc/swagger.yml \
		-t ./internal/generated -C ./swagger-gen/default-server.yml \
		--template-dir ./swagger-gen/templates \
		--name ${PROJECT}
# Генерация собственного клиента
gen-serv-client:
	@rm -rf ./pkg/${PROJECT}-client/client
	@rm -rf ./pkg/${PROJECT}-client/models
	@mkdir -p ./pkg/${PROJECT}-client
	@${SWAGGER_BIN} generate client --allow-template-override \
    		-f ./swagger-doc/swagger.yml \
    		--template-dir ./swagger-gen/templates \
            --name ${PROJECT} \
    		-C ./swagger-gen/default-client.yml \
    		-t ./pkg/${PROJECT}-client
	@cd ./pkg/${PROJECT}-client;go mod init;go mod tidy

gen-clients:
	@echo "Generating client..."

swagger-version:
	@echo "${SWAGGER_PLATFORM}"
	@${SWAGGER_BIN} version

# Удаление директорий со сгенерированным кодом
clean-gen:
	@echo "Removing generated files..."
	@rm -rf ./internal/generated
	@rm -rf ./pkg/products-client/client
	@rm -rf ./pkg/products-client/models

## Development
install-tools: install-lint install-swagger

lint: install-lint
	@echo "Running golangci-lint..."
	@PATH=$(PATH); ${GOLANGCI_BIN} run

test:
	@echo "Running test..."
	@go test ./... -cover -short -count=1

test-all:
	@echo "Running test..."
	@go test ./... -cover -count=1

## Docker
build:
	@echo "Building docker image..."
	@echo "platform ${SWAGGER_PLATFORM}"
	docker build --ssh default --no-cache \
		--build-arg SWAGGER_PLATFORM=${SWAGGER_PLATFORM} \
		--build-arg VERSION=${VERSION} \
		--build-arg BUILD_TIME=${BUILD_TIME} \
		--build-arg APP_NAME=${PROJECT} \
		-t ${PROJECT} .
## Docker
build-dev:
	@echo "Building docker image..."
	@echo "platform ${SWAGGER_PLATFORM}"
	docker build --ssh default --no-cache \
		--build-arg SWAGGER_PLATFORM=${SWAGGER_PLATFORM} \
		--build-arg VERSION=${VERSION} \
		--build-arg BUILD_TIME=${BUILD_TIME} \
		--build-arg APP_NAME=${PROJECT} \
		--target builder \
		-t ${PROJECT}:dev .


vendor:
	@go mod vendor

infra-up:
	@echo "starting infra version: ${PROJECT}"
	@docker-compose -f docker-compose-infra.yml -p ${PROJECT} up -d

infra-ps:
	@docker-compose -f docker-compose-infra.yml -p ${PROJECT} ps
infra-logs:
	@docker-compose -f docker-compose-infra.yml -p ${PROJECT} logs -f

infra-down:
	@echo "down infra version: ${PROJECT}"
	@docker-compose  -f docker-compose-infra.yml -p ${PROJECT} down -v

migadd: install-goose
	#make migadd name=test
	@mkdir -p database/schemas
	@$(GOOSE_BIN) -dir database/schemas create $(name) sql

gensql: install-sqlc
	@$(SQLC_BIN) generate -f database/sqlc.yaml
