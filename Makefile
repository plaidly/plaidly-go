.PHONY: generate test tidy

# Pinned oapi-codegen version — keep in sync with plaidly-api.
OAPI_CODEGEN_VERSION ?= v2.4.1
SPEC ?= spec/openapi.yaml

generate:
	@mkdir -p generated/plaidlyapi
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@$(OAPI_CODEGEN_VERSION) \
		-config oapi-cfg.yaml $(SPEC)
	go mod tidy

tidy:
	go mod tidy

test:
	go test ./...
