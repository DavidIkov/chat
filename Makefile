# afona_chat — common developer tasks.
#
# Layout:
#   cmd/api_server      -> JSON HTTP API backend (Postgres)
#   cmd/webui_server    -> server-side-rendered HTML frontend (proxy)

GO         ?= go
API_ADDR   ?= :8080
WEBUI_ADDR ?= :8000
DB_URL     ?= postgres://admin:admin@localhost:7337/chatdb?sslmode=disable

GOFILES := $(shell find . -name '*.go' -not -path './vendor/*')

.PHONY: all build vet fmt fmt-check check run-api run-webui dev down clean

all: check

## build: compile every package and binary
build:
	$(GO) build ./...

## vet: run go vet
vet:
	$(GO) vet ./...

## fmt: format all Go source in place
fmt:
	gofmt -w $(GOFILES)

## fmt-check: fail if any Go file is not gofmt-clean
fmt-check:
	@out="$$(gofmt -l $(GOFILES))"; \
	if [ -n "$$out" ]; then echo "not gofmt-clean:"; echo "$$out"; exit 1; fi

## check: fmt-check + vet + build
check: fmt-check vet build

## run-api: run the api_server (needs Postgres; start it with `make dev`)
run-api:
	$(GO) run ./cmd/api_server -listenURL $(API_ADDR) -dbURL "$(DB_URL)"

## run-webui: run the webui_server
run-webui:
	$(GO) run ./cmd/webui_server -listenURL $(WEBUI_ADDR)

## dev: start the local Postgres (docker compose)
dev:
	cd dev && docker compose up

## down: stop the local Postgres
down:
	cd dev && docker compose down

## clean: remove cached build artifacts
clean:
	$(GO) clean ./...
