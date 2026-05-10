.PHONY: build test bench lint release clean

build: ## Build the ofdir binary
	go build -o bin/ofdir ./cmd/ofdir

test: ## Run tests
	go test ./... -race -count=1

bench: ## Run benchmarks
	go test ./... -bench=. -benchmem

lint: ## Run static analysis
	golangci-lint run

release: ## Build release with goreleaser
	goreleaser release --clean

clean: ## Remove build artifacts
	rm -rf bin/
