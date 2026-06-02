default: fmt vet lint install generate

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

generate:
	cd tools; go generate ./...

fmt:
	gofmt -s -w -e .

validate: fmt vet lint

test: s=resources
test:
	go test -v -cover -timeout=120s -parallel=10 ./internal/$(s)/...

coverage:
	go test -v -coverprofile=coverage.out -covermode=atomic -coverpkg=./internal/... -timeout=120s -parallel=10 ./internal/resources/... ./internal/data-sources/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Unit test coverage report written to coverage.html"

testacc:
	TF_ACC=1 go test -v -count=1 -cover -p 1 -timeout 120m ./internal/tests/...

coverageacc:
	TF_ACC=1 go test -v -coverprofile=coverage-acc.out -covermode=atomic -coverpkg=./internal/resources/...,./internal/data-sources/...,./internal/provider/... -count=1 -p 1 -timeout 120m ./internal/tests/...
	go tool cover -html=coverage-acc.out -o coverage-acc.html
	@echo "Acceptance test coverage report written to coverage-acc.html"

.PHONY: fmt vet lint test testacc coverage coverageacc build install generate

vet:
	go vet -v ./...