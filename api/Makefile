.DEFAULT_GOAL := help
BINARY := bin/api
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

.PHONY: help
help: ## Tampilkan daftar perintah
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

.PHONY: setup
setup: ## Siapkan .env dan nyalakan MySQL + Redis
	@test -f .env || (cp .env.example .env && echo "  .env dibuat dari .env.example")
	@grep -q '^JWT_SECRET=.\{32,\}' .env || echo "  ISI DULU JWT_SECRET di .env (pakai: make secret)"
	docker compose up -d

.PHONY: secret
secret: ## Generate JWT_SECRET acak
	@openssl rand -base64 48 | tr -d '\n'; echo

.PHONY: run
run: ## Jalankan API (migrasi jalan otomatis saat start)
	go run ./cmd/api

.PHONY: build
build: ## Build binary statis ke bin/api
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/api
	@echo "  $(BINARY) ($(VERSION))"

.PHONY: migrate
migrate: ## Jalankan migrasi lalu keluar
	go run ./cmd/api -migrate

.PHONY: migrate-down
migrate-down: ## Mundurkan satu versi migrasi
	go run ./cmd/api -migrate-down

.PHONY: migrate-status
migrate-status: ## Tampilkan status migrasi
	go run ./cmd/api -migrate-status

.PHONY: test
test: ## Jalankan test
	go test ./... -race -count=1

.PHONY: check
check: ## gofmt + vet + test
	@test -z "$$(gofmt -l . | tee /dev/stderr)" || (echo "  ada file belum diformat, jalankan: gofmt -w ."; exit 1)
	go vet ./...
	go test ./... -race -count=1

.PHONY: up
up: ## Nyalakan MySQL + Redis
	docker compose up -d

.PHONY: down
down: ## Matikan MySQL + Redis
	docker compose down

.PHONY: clean
clean: ## Hapus hasil build
	rm -rf bin/
