#!/usr/bin/make -f
include build/scripts/cosmos.mk build/scripts/constants.mk

# Specify the default target if none is provided
.DEFAULT_GOAL := build

###############################################################################
###                                  Build                                  ###
###############################################################################

BUILD_TARGETS := build install

build: BUILD_ARGS=-o $(OUT_DIR)/

build-linux-amd64:
	GOOS=linux GOARCH=amd64 LEDGER_ENABLED=false $(MAKE) build

build-linux-arm64:
	GOOS=linux GOARCH=arm64 LEDGER_ENABLED=false $(MAKE) build

$(BUILD_TARGETS): forge-build sync $(OUT_DIR)/
	@echo "Building ${TESTAPP_DIR}"
	@cd ${CURRENT_DIR}/$(TESTAPP_DIR) && go $@ -mod=readonly $(BUILD_FLAGS) $(BUILD_ARGS) ./...

$(OUT_DIR)/:
	mkdir -p $(OUT_DIR)/

build-clean: 
	@$(MAKE) clean build

clean:
	@rm -rf .tmp/ 
	@rm -rf $(OUT_DIR)
	@$(MAKE) forge-clean

#################
#     forge     #
#################

forge-build: |
	@forge build --extra-output-files bin --extra-output-files abi  --root $(CONTRACTS_DIR)

forge-clean: |
	@forge clean --root $(CONTRACTS_DIR)


#################
#     proto     #
#################

protoImageName    := "ghcr.io/cosmos/proto-builder"
protoImageVersion := "0.14.0"

proto:
	@$(MAKE) buf-lint-fix buf-lint proto-build

proto-build:
	@docker run --rm -v ${CURRENT_DIR}:/workspace --workdir /workspace $(protoImageName):$(protoImageVersion) sh ./build/scripts/proto_generate.sh

###############################################################################
###                                 Docker                                  ###
###############################################################################

# Variables
DOCKER_TYPE ?= base
ARCH ?= arm64
GO_VERSION ?= 1.21.6
IMAGE_NAME ?= beacond
IMAGE_VERSION ?= v0.0.0
BASE_IMAGE ?= beacond/base:$(IMAGE_VERSION)

# Docker Paths
BASE_DOCKER_PATH = ./app/docker
EXEC_DOCKER_PATH = $(BASE_DOCKER_PATH)/base.Dockerfile
LOCAL_DOCKER_PATH = $(BASE_DOCKER_PATH)/local/Dockerfile
SEED_DOCKER_PATH =  $(BASE_DOCKER_PATH)/seed/Dockerfile
VAL_DOCKER_PATH =  $(BASE_DOCKER_PATH)/validator/Dockerfile
LOCALNET_CLIENT_PATH = ./e2e/precompile/beacond
LOCALNET_DOCKER_PATH = $(LOCALNET_CLIENT_PATH)/Dockerfile

# Image Build
docker-build:
	@echo "Build a release docker image for the Cosmos SDK chain..."
	@$(MAKE) docker-build-$(DOCKER_TYPE)

# Docker Build Types
docker-build-base:
	$(call docker-build-helper,$(EXEC_DOCKER_PATH),base)

docker-build-local:
	$(call docker-build-helper,$(LOCAL_DOCKER_PATH),local,--build-arg BASE_IMAGE=$(BASE_IMAGE))

docker-build-seed:
	$(call docker-build-helper,$(SEED_DOCKER_PATH),seed,--build-arg BASE_IMAGE=$(BASE_IMAGE))

docker-build-validator:
	$(call docker-build-helper,$(VAL_DOCKER_PATH),validator,--build-arg BASE_IMAGE=$(BASE_IMAGE))

docker-build-localnet:
	$(call docker-build-helper,$(LOCALNET_DOCKER_PATH),localnet,--build-arg BASE_IMAGE=$(BASE_IMAGE))

# Docker Build Function
define docker-build-helper
	docker build \
	--build-arg GO_VERSION=$(GO_VERSION) \
	--platform linux/$(ARCH) \
	--build-arg GIT_COMMIT=$(shell git rev-parse HEAD) \
	--build-arg GIT_BRANCH=$(shell git rev-parse --abbrev-ref HEAD) \
	--build-arg GOOS=linux \
	--build-arg GOARCH=$(ARCH) \
	-f $(1) \
	-t $(IMAGE_NAME)/$(2):$(IMAGE_VERSION) \
	$(if $(3),$(3)) \
	.

endef

.PHONY: docker-build-localnet

###############################################################################
###                                 CodeGen                                 ###
###############################################################################

generate:
	@$(MAKE) abigen-install mockery 
	@for module in $(MODULES); do \
		echo "Running go generate in $$module"; \
		(cd $$module && go generate ./...) || exit 1; \
	done
	@$(MAKE) sszgen

abigen-install:
	@echo "--> Installing abigen"
	@go install github.com/ethereum/go-ethereum/cmd/abigen@latest

mockery-install:
	@echo "--> Installing mockery"
	@go install github.com/vektra/mockery/v2@latest

mockery:
	@$(MAKE) mockery-install
	@echo "Running mockery..."
	@mockery


###############################################################################
###                           Tests & Simulation                            ###
###############################################################################

#################
#    beacond     #
#################

# TODO: add start-erigon and start-nethermind

# Start beacond
start:
	@./app/entrypoint.sh

# Start reth node
start-reth:
	@rm -rf .tmp/eth-home
	@docker run \
	-p 30303:30303 \
	-p 8545:8545 \
	-p 8551:8551 \
	--rm -v $(PWD)/app:/app \
	-v $(PWD)/.tmp:/.tmp \
	ghcr.io/paradigmxyz/reth node \
	--chain ./app/eth-genesis.json \
	--http \
	--http.addr "0.0.0.0" \
	--http.api eth \
	--authrpc.addr "0.0.0.0" \
	--authrpc.jwtsecret ./app/jwt.hex
	--datadir .tmp/reth
	
# Init and start geth node
start-geth:
	rm -rf .tmp/geth
	docker run \
	--rm -v $(PWD)/app:/app \
	-v $(PWD)/.tmp:/.tmp \
	ethereum/client-go init \
	--datadir .tmp/geth \
	./app/eth-genesis.json
	docker run \
	-p 30303:30303 \
	-p 8545:8545 \
	-p 8551:8551 \
	--rm -v $(PWD)/app:/app \
	-v $(PWD)/.tmp:/.tmp \
	ethereum/client-go \
	--http \
	--http.addr 0.0.0.0 \
	--http.api eth \
	--authrpc.addr 0.0.0.0 \
	--authrpc.jwtsecret ./app/jwt.hex \
	--authrpc.vhosts "*" \
	--datadir .tmp/geth

# Start nethermind node
start-nethermind:
	docker run \
	-p 30303:30303 \
	-p 8545:8545 \
	-p 8551:8551 \
	-v $(PWD)/app:/app \
	-v $(PWD)/.tmp:/.tmp \
	nethermind/nethermind \
	--JsonRpc.Port 8545 \
	--JsonRpc.EngineEnabledModules "eth,net,engine" \
	--JsonRpc.EnginePort 8551 \
	--JsonRpc.EngineHost 0.0.0.0 \
	--JsonRpc.Host 0.0.0.0 \
	--JsonRpc.JwtSecretFile ../app/jwt.hex \
	--Sync.PivotNumber 0 \
	--Init.ChainSpecPath ../app/eth-nether-genesis.json

# Start besu node
start-besu:
	docker run \
	-p 30303:30303 \
	-p 8545:8545 \
	-p 8551:8551 \
	-v $(PWD)/app:/app \
	-v $(PWD)/.tmp:/.tmp \
	hyperledger/besu:latest \
	--data-path=.tmp/besu \
	--genesis-file=../../app/eth-genesis.json \
	--rpc-http-enabled \
	--rpc-http-api=ETH,NET,ENGINE,DEBUG,NET,WEB3 \
	--host-allowlist="*" \
	--rpc-http-cors-origins="all" \
	--engine-rpc-port=8551 \
	--engine-rpc-enabled \
	--engine-host-allowlist="*" \
	--engine-jwt-secret=../../app/jwt.hex



#################
#     unit      #
#################


test-unit:
	@$(MAKE) forge-test
	@echo "Running unit tests..."
	go test ./...

test-unit-cover:
	@$(MAKE) forge-test
	@echo "Running unit tests with coverage..."
	go test -race -coverprofile=coverage-test-unit-cover.txt -covermode=atomic ./...

#################
#     forge     #
#################

forge-test:
	@echo "Running forge test..."
	@forge test --root $(CONTRACTS_DIR)

#################
#      e2e      #
#################

test-e2e:
	@$(MAKE) test-e2e-no-build

test-e2e-no-build:
	@echo "Running e2e tests..."

###############################################################################
###                                Linting                                  ###
###############################################################################

format:
	@$(MAKE) license-fix buf-lint-fix forge-lint-fix golangci-fix

lint:
	@$(MAKE) license buf-lint forge-lint golangci

#################
#     forge     #
#################

forge-lint-fix:
	@echo "--> Running forge fmt"
	@cd $(CONTRACTS_DIR) && forge fmt

forge-lint:
	@echo "--> Running forge lint"
	@cd $(CONTRACTS_DIR) && forge fmt --check

#################
# golangci-lint #
#################

golangci_version=v1.55.2

golangci-install:
	@echo "--> Installing golangci-lint $(golangci_version)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(golangci_version)

golangci:
	@$(MAKE) golangci-install
	@echo "--> Running linter"
	@go list -f '{{.Dir}}/...' -m | xargs golangci-lint run  --timeout=10m --concurrency 8 -v 

golangci-fix:
	@$(MAKE) golangci-install
	@echo "--> Running linter"
	@go list -f '{{.Dir}}/...' -m | xargs golangci-lint run  --timeout=10m --fix --concurrency 8 -v 


#################
#    license    #
#################

license-install:
	@echo "--> Installing google/addlicense"
	@go install github.com/google/addlicense

license:
	@$(MAKE) license-install
	@echo "--> Running addlicense with -check"
	@for module in $(MODULES); do \
		(cd $$module && addlicense -check -v -f ./LICENSE.header ./.) || exit 1; \
	done

license-fix:
	@$(MAKE) license-install
	@echo "--> Running addlicense"
	@for module in $(MODULES); do \
		(cd $$module && addlicense -v -f ./LICENSE.header ./.) || exit 1; \
	done


#################
#     gosec     #
#################

gosec-install:
	@echo "--> Installing gosec"
	@go install github.com/cosmos/gosec/v2/cmd/gosec

gosec:
	@$(MAKE) gosec-install
	@echo "--> Running gosec"
	@gosec ./...


#################
#     proto     #
#################

protoDir := "proto"

buf-install:
	@echo "--> Installing buf"
	@go install github.com/bufbuild/buf/cmd/buf

buf-lint-fix:
	@$(MAKE) buf-install 
	@echo "--> Running buf format"
	@buf format -w --error-format=json $(protoDir)

buf-lint:
	@$(MAKE) buf-install 
	@echo "--> Running buf lint"
	@buf lint --error-format=json $(protoDir)


#################
#    sszgen    #
#################

sszgen-install:
	@echo "--> Installing sszgen"
	@go install github.com/prysmaticlabs/fastssz/sszgen


SSZ_STRUCTS=BeaconBlockData

sszgen:
	@$(MAKE) sszgen-install
	@echo "--> Running sszgen on all structs with ssz tags"
	@sszgen -path ./types/consensus/v1/ -objs ${SSZ_STRUCTS}
###############################################################################
###                             Dependencies                                ###
###############################################################################

tidy: |
	go mod tidy

repo-rinse: |
	git clean -xfd
	git submodule foreach --recursive git clean -xfd
	git submodule foreach --recursive git reset --hard
	git submodule update --init --recursive


.PHONY: build build-linux-amd64 build-linux-arm64 \
	$(BUILD_TARGETS) $(OUT_DIR)/ build-clean clean \
	forge-build forge-clean proto proto-build docker-build \
	docker-build-base docker-build-local docker-build-seed \
	docker-build-validator docker-build-localnet generate \
	abigen-install mockery-install mockery \
	start test-unit test-unit-race test-unit-cover forge-test \
	test-e2e test-e2e-no-build hive-setup hive-view test-hive \
	test-hive-v test-localnet test-localnet-no-build format lint \
	forge-lint-fix forge-lint golangci-install golangci golangci-fix \
	license-install license license-fix \
	gosec-install gosec buf-install buf-lint-fix buf-lint sync tidy repo-rinse
