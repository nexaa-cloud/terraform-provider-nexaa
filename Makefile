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

testacc:
	TF_ACC=1 go test -v -count=1 -cover -p 1 -timeout 120m ./...

.PHONY: fmt vet lint test testacc build install generate

vet:
	go vet -v ./...