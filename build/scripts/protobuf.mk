#!/usr/bin/make -f

protoImageName    := "ghcr.io/cosmos/proto-builder"
protoImageVersion := "0.14.0"
modulesProtoDir := "beacond/x/beacon/proto"

## Protobuf:
proto: ## run all the proto tasks
	@$(MAKE) buf-lint-fix buf-lint proto-build

proto-build: ## build the proto files
	@docker run --rm -v ${CURRENT_DIR}:/workspace --workdir /workspace $(protoImageName):$(protoImageVersion) sh ./build/scripts/proto_generate_pulsar.sh

proto-clean: ## clean the proto files
	@find . -name '*.pb.go' -delete
	@find . -name '*.pb.gw.go' -delete
	
buf-install:
	@echo "--> Installing buf"
	@go install github.com/bufbuild/buf/cmd/buf

buf-lint-fix:
	@$(MAKE) buf-install 
	@echo "--> Running buf format"
	@buf format -w --error-format=json $(modulesProtoDir)

buf-lint:
	@$(MAKE) buf-install 
	@echo "--> Running buf lint"
	@buf lint --error-format=json $(modulesProtoDir)