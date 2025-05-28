PROTO_DIR = proto
BUSINESS_BOT_DIR = business_bot
CREATOR_BOT_DIR = creator_bot

PROTO_FILES = $(wildcard $(PROTO_DIR)/*.proto)

PROTOC = protoc
PROTOC_GEN_GO = protoc-gen-go
PROTOC_GEN_GO_GRPC = protoc-gen-go-grpc

GO_MODULE_BUSINESS = ssuspy-bot
GO_MODULE_CREATOR = ssuspy-creator-bot

.PHONY: all
all: generate

.PHONY: generate
generate: generate-business generate-creator

.PHONY: generate-business
generate-business: check-tools
	@echo "Generating gRPC server code for business_bot..."
	@mkdir -p $(BUSINESS_BOT_DIR)/pb
	$(PROTOC) \
		--go_out=$(BUSINESS_BOT_DIR)/pb \
		--go_opt=paths=source_relative \
		--go_opt=Mbot.proto=. \
		--go-grpc_out=$(BUSINESS_BOT_DIR)/pb \
		--go-grpc_opt=paths=source_relative \
		--go-grpc_opt=Mbot.proto=. \
		--proto_path=$(PROTO_DIR) \
		$(PROTO_FILES)

.PHONY: generate-creator
generate-creator: check-tools
	@echo "Generating gRPC client code for creator_bot..."
	@mkdir -p $(CREATOR_BOT_DIR)/pb
	$(PROTOC) \
		--go_out=$(CREATOR_BOT_DIR)/pb \
		--go_opt=paths=source_relative \
		--go_opt=Mbot.proto=. \
		--go-grpc_out=$(CREATOR_BOT_DIR)/pb \
		--go-grpc_opt=paths=source_relative \
		--go-grpc_opt=Mbot.proto=. \
		--proto_path=$(PROTO_DIR) \
		$(PROTO_FILES)

.PHONY: check-tools
check-tools:
	@which $(PROTOC) > /dev/null || (echo "Error: protoc is not installed" && exit 1)
	@which $(PROTOC_GEN_GO) > /dev/null || (echo "Error: protoc-gen-go is not installed. Run: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest" && exit 1)
	@which $(PROTOC_GEN_GO_GRPC) > /dev/null || (echo "Error: protoc-gen-go-grpc is not installed. Run: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest" && exit 1)

.PHONY: install-tools
install-tools:
	@echo "Installing protoc-gen-go and protoc-gen-go-grpc..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
