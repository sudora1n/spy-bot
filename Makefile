BUF = buf

.PHONY: all generate
all: generate

.PHONY: generate
generate:
	@which $(BUF) > /dev/null || (echo "Error: buf CLI is not installed" && exit 1)
	@echo "Generating code via Buf v2..."
	$(BUF) generate --template buf.gen.yaml
