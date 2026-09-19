include .env
export

# Go configuration
GO := go
MAIN := ./cmd

# Docker Compose configuration
DOCKER_COMPOSE := docker compose
COMPOSE_FILE := infra/docker-compose.yml

# Database configuration
DB_DSN := postgres://$(MIGRATION_USER):$(MIGRATION_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=$(POSTGRES_SSLMODE)
MIGRATIONS_DIR := ./migrations

# Phony targets
.PHONY: run up down migrate_up migrate_down lint \
	test_all test_unit test_admission test_scheduler \
	test_integration test_integration_all test_integration_stress \
	test_admission_integration test_scheduler_integration\
	test_scheduler_integration test_job test_recovery  \
	test_worker test_concurrency 

run: ## Run the main Go application
	$(GO) run $(MAIN)

up: ## Start the development environment via Docker Compose (detached)
	$(DOCKER_COMPOSE) --env-file .env -f $(COMPOSE_FILE) up -d
 
down: ## Tear down containers and networks cleanly
	$(DOCKER_COMPOSE) -f $(COMPOSE_FILE) down

migrate_up: ## Run database migrations up
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_DSN)" up

migrate_down: ## Roll back database migrations
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_DSN)" down

lint:
	golangci-lint run ./...

test_admission: ## Run admission unit tests
	$(GO) test ./internal/admission/usecase -count=1

test_scheduler: ## Run scheduler unit tests
	$(GO) test ./internal/scheduler -count=1

test_recovery: ## Run the focused recovery integration suite
	$(GO) test ./tests/integration/recovery -count=1 -v

test_worker: ## Run the focused worker integration suite
	$(GO) test ./tests/integration/worker -count=1 -v

test_scheduler_integration: ## Run the scheduler integration suite
	$(GO) test ./tests/integration/scheduler -count=1 -v

test_job: ## Run the job integration suite
	$(GO) test ./tests/integration/job -count=1 -v

test_admission_integration: ## Run the admission integration suite
	$(GO) test ./tests/integration/admission -count=1 -v

test_concurrency: ## Run all concurrency integration tests
	$(GO) test ./tests/concurrency -count=1 -v

test_integration: test_integration_all ## Run all integration tests once (compatibility alias)

test_integration_all: ## Run all integration tests once
	$(GO) test ./tests/integration/... -count=1

test_integration_stress: ## Repeat all integration suites to expose flaky behavior
	$(GO) test ./tests/integration/... -count=20 -v

test_unit: ## Run all internal unit tests
	$(GO) test ./internal/... -count=1

test_all: ## Run all unit, integration, and concurrency tests
	$(GO) test ./internal/... ./tests/integration/... ./tests/concurrency/... -count=1
