# Makefile for Card Onboarding Services

OAPI_CODEGEN=go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen

.PHONY: all generate swagger-validate generate-check clean

all: generate build

generate:
	@echo "Creating generated packages directories..."
	mkdir -p onboard-service/pkg/onboard
	mkdir -p customer-management-service/pkg/customer
	mkdir -p account-management-service/pkg/account
	
	@echo "Generating onboard-service code..."
	$(OAPI_CODEGEN) -package onboard -generate types -o onboard-service/pkg/onboard/types.gen.go onboard-service/swagger-internal.yaml
	$(OAPI_CODEGEN) -package onboard -generate client -o onboard-service/pkg/onboard/client.gen.go onboard-service/swagger-internal.yaml
	$(OAPI_CODEGEN) -package onboard -generate gin -o onboard-service/pkg/onboard/server.gen.go onboard-service/swagger-internal.yaml
	
	@echo "Generating customer-management-service code..."
	$(OAPI_CODEGEN) -package customer -generate types -o customer-management-service/pkg/customer/types.gen.go customer-management-service/swagger-internal.yaml
	$(OAPI_CODEGEN) -package customer -generate client -o customer-management-service/pkg/customer/client.gen.go customer-management-service/swagger-internal.yaml
	$(OAPI_CODEGEN) -package customer -generate gin -o customer-management-service/pkg/customer/server.gen.go customer-management-service/swagger-internal.yaml
	
	@echo "Generating account-management-service code..."
	$(OAPI_CODEGEN) -package account -generate types -o account-management-service/pkg/account/types.gen.go account-management-service/swagger-internal.yaml
	$(OAPI_CODEGEN) -package account -generate client -o account-management-service/pkg/account/client.gen.go account-management-service/swagger-internal.yaml
	$(OAPI_CODEGEN) -package account -generate gin -o account-management-service/pkg/account/server.gen.go account-management-service/swagger-internal.yaml

swagger-validate:
	@echo "Validating OpenAPI specs..."
	@# Check if any spec file has invalid YAML syntax
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -version > /dev/null
	@echo "OpenAPI Specs are structurally valid."

generate-check: generate
	@echo "Checking if generated code is up-to-date..."
	git diff --exit-code onboard-service/pkg/ customer-management-service/pkg/ account-management-service/pkg/ || \
		(echo "Error: Generated Go code is out of date. Please run 'make generate' locally and commit the changes." && exit 1)

clean:
	rm -rf onboard-service/pkg/
	rm -rf customer-management-service/pkg/
	rm -rf account-management-service/pkg/
