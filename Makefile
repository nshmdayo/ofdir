.PHONY: build test bench lint release clean container-build dev-shell

CONTAINER_RUNTIME ?= podman
CONTAINER_IMAGE   ?= sd-dev
GOMODCACHE_VOL    ?= sd-gomodcache
USE_CONTAINER     ?= 1

PODMAN_RUN_OPTS = --rm \
	--userns=keep-id \
	-v "$(CURDIR):/app:Z" \
	-v "$(GOMODCACHE_VOL):/go/pkg/mod:Z" \
	-w /app

ifeq ($(USE_CONTAINER),1)
RUNNER = $(CONTAINER_RUNTIME) run $(PODMAN_RUN_OPTS) $(CONTAINER_IMAGE)
else
RUNNER =
endif

build: ## Build the ofdir binary
	$(RUNNER) go build -o bin/ofdir ./cmd/ofdir

test: ## Run tests
	$(RUNNER) go test ./... -race -count=1

bench: ## Run benchmarks
	$(RUNNER) go test ./... -bench=. -benchmem

lint: ## Run static analysis
	$(RUNNER) golangci-lint run

release: ## Build release with goreleaser
	goreleaser release --clean

clean: ## Remove build artifacts
	rm -rf bin/

container-build: ## Build the development container image
	$(CONTAINER_RUNTIME) build -f Containerfile -t $(CONTAINER_IMAGE) .

dev-shell: ## Start an interactive shell in the development container
	$(CONTAINER_RUNTIME) run -it $(PODMAN_RUN_OPTS) $(CONTAINER_IMAGE) bash
