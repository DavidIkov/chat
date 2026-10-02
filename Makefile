# chat — common developer tasks.
#
# Layout:
#   cmd/api_server      -> JSON HTTP API backend (Postgres)
#   cmd/webui_server    -> server-side-rendered HTML frontend (proxy)

GO         ?= go
PYTHON     ?= python3
API_ADDR   ?= :8080
WEBUI_ADDR ?= :8000
# Optional explicit config file for `make run-gui` (defaults to
# $XDG_CONFIG_HOME/chat/gui_client.json).
GUI_CONFIG ?=
DB_URL     ?= postgres://admin:admin@localhost:7337/chatdb?sslmode=disable
# The integration tests create and drop their own throwaway database, so this
# must point at a database the role may connect to for maintenance (the compose
# `postgres` database works) and that role must be allowed to CREATE DATABASE.
TEST_DB_URL ?= postgres://admin:admin@localhost:7338/postgres?sslmode=disable

GOFILES := $(shell find . -name '*.go' -not -path './vendor/*')

.PHONY: all build vet fmt fmt-check check run-api run-webui run-gui test test-short test-gui pgup pgdown clean

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

## test: start the throwaway test Postgres, run the full suite, then stop it
# Backed by dev/docker-compose.test.yml, which mounts no volume, so the database
# is discarded when the container is torn down (whether the tests pass or fail).
test:
	@set -e; \
	trap 'docker compose -f dev/docker-compose.test.yml down -v' EXIT; \
	docker compose -f dev/docker-compose.test.yml up -d --wait; \
	TEST_DB_URL="$(TEST_DB_URL)" $(GO) test -count=1 ./...

## test-short: run only the tests that do not need a database
test-short:
	$(GO) test -short ./...

## test-gui: run the GUI client's headless unit tests (no display needed)
test-gui:
	cd gui_client && $(PYTHON) -m unittest discover -s tests -v

## run-api: run the api_server (needs Postgres; start it with `make pgup`)
run-api:
	$(GO) run ./cmd/api_server -listenURL $(API_ADDR) -dbURL "$(DB_URL)"

## run-webui: run the webui_server
run-webui:
	$(GO) run ./cmd/webui_server -listenURL $(WEBUI_ADDR)

## run-gui: run the desktop GUI client (needs Python 3.10+, python3-tk and a display)
run-gui:
	cd gui_client && $(PYTHON) -m chat_client $(if $(GUI_CONFIG),--config "$(GUI_CONFIG)")

## pgup: start the local Postgres (docker compose, foreground)
pgup:
	cd dev && docker compose up

pgdown:
	cd dev && docker compose down

## clean: remove cached build artifacts
clean:
	$(GO) clean ./...
