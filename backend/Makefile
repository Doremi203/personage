DB = postgres://user:pass@localhost:5432/
DB_PARAMS = ?sslmode=disable

export GOOSE_DRIVER=postgres
MIGRATE_CREATE = goose create $(name) sql -s
export TESTCONTAINERS_RYUK_DISABLED=true
export BASE_MIGRATIONS_PATH=$(PWD)

deps: deps/migrate deps/go-test-coverage deps/buf deps/mockgen deps/protoc deps/linters

deps/migrate:
	go install github.com/pressly/goose/v3/cmd/goose@latest

deps/go-test-coverage:
	go install github.com/vladopajic/go-test-coverage/v2@latest

deps/buf:
	go install github.com/bufbuild/buf/cmd/buf@latest

deps/mockgen:
	go install go.uber.org/mock/mockgen@latest

deps/protoc:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
	go install github.com/envoyproxy/protoc-gen-validate/cmd/protoc-gen-validate-go@latest

deps/linters:
	brew install golangci-lint

common/pull/proto: common/pull/proto/googleapis common/pull/proto/protoc-gen-openapiv2 common/pull/proto/validate

common/pull/proto/googleapis:
	$(call CLONE_PROTO_REPO, https://github.com/googleapis/googleapis,google/api/http.proto google/api/annotations.proto google/api/field_behavior.proto)

common/pull/proto/protoc-gen-openapiv2:
	$(call CLONE_PROTO_REPO, https://github.com/grpc-ecosystem/grpc-gateway,protoc-gen-openapiv2/options/annotations.proto protoc-gen-openapiv2/options/openapiv2.proto)

common/pull/proto/validate:
	$(call CLONE_PROTO_REPO, https://github.com/bufbuild/protoc-gen-validate,validate/validate.proto)

define CLONE_PROTO_REPO
	@echo "Клонируем $(1)"
	@rm -rf tmp
	@git clone --filter=blob:none --no-checkout $(1) tmp
	@cd tmp && \
		git sparse-checkout init --no-cone && \
		git sparse-checkout set $(2) && \
		git checkout
	@mkdir -p libs/proto
	@cd tmp && \
		rsync -aR $(2) $(CURDIR)/libs/proto/
	@rm -rf tmp
endef

generate: tasker/generate notificator/generate auth/proto/generate

auth/proto/generate:
	cd libs/go/auth && go generate ./...

services/deploy:
	docker compose up

lint:
	golangci-lint run

# Tasker
tasker/run: tasker/generate
	docker compose up tasker -d --build

tasker/generate:
	cd tasker && go generate ./...

TASKER_MIGRATIONS = tasker/migrations

tasker/migrate/up:
	goose -v -dir "$(TASKER_MIGRATIONS)" "$(DB)tasker$(DB_PARAMS)" up

tasker/migrate/down:
	goose -v -dir "$(TASKER_MIGRATIONS)" "$(DB)tasker$(DB_PARAMS)" reset

tasker/migrate/down-one:
	goose -v -dir "$(TASKER_MIGRATIONS)" "$(DB)tasker$(DB_PARAMS)" down

tasker/migrate/create:
	cd $(TASKER_MIGRATIONS) && $(MIGRATE_CREATE)

tasker/create-test-tasks:
	cd tasker/scripts && go run create_test_tasks.go

# Notificator
notificator/run: notificator/generate
	docker compose up notificator -d --build

notificator/generate:
	cd notificator && go generate ./...

NOTIFICATOR_MIGRATIONS = notificator/migrations

notificator/migrate/up:
	goose -v -dir "$(NOTIFICATOR_MIGRATIONS)" "$(DB)notificator$(DB_PARAMS)" up

notificator/migrate/down:
	goose -v -dir "$(NOTIFICATOR_MIGRATIONS)" "$(DB)notificator$(DB_PARAMS)" reset

notificator/migrate/down-one:
	goose -v -dir "$(NOTIFICATOR_MIGRATIONS)" "$(DB)notificator$(DB_PARAMS)" down

notificator/migrate/create:
	cd $(NOTIFICATOR_MIGRATIONS) && $(MIGRATE_CREATE)

# Auth
AUTH_MIGRATIONS = Personage.Auth/migrations

auth/migrate/up:
	goose -v -dir "$(AUTH_MIGRATIONS)" "$(DB)auth$(DB_PARAMS)" up

auth/migrate/down:
	goose -v -dir "$(AUTH_MIGRATIONS)" "$(DB)auth$(DB_PARAMS)" reset

auth/migrate/down-one:
	goose -v -dir "$(AUTH_MIGRATIONS)" "$(DB)auth$(DB_PARAMS)" down

auth/migrate/create:
	cd $(AUTH_MIGRATIONS) && $(MIGRATE_CREATE)

# Traitex
TRAITEX_MIGRATIONS = traitex/migrations

traitex/migrate/up:
	goose -v -dir "$(TRAITEX_MIGRATIONS)" "$(DB)traitex$(DB_PARAMS)" up

traitex/migrate/down:
	goose -v -dir "$(TRAITEX_MIGRATIONS)" "$(DB)traitex$(DB_PARAMS)" reset

traitex/migrate/down-one:
	goose -v -dir "$(TRAITEX_MIGRATIONS)" "$(DB)traitex$(DB_PARAMS)" down

traitex/migrate/create:
	cd $(TRAITEX_MIGRATIONS) && $(MIGRATE_CREATE)

tasker-eval/migrate/up:
	goose -v -dir "$(TASKER_MIGRATIONS)" "$(DB)tasker_eval$(DB_PARAMS)" up

tasker-eval/migrate/down:
	goose -v -dir "$(TASKER_MIGRATIONS)" "$(DB)tasker_eval$(DB_PARAMS)" reset

tasker-eval/migrate/reset: tasker-eval/migrate/down tasker-eval/migrate/up

tasker-eval/run:
	docker compose -f docker-compose.yml -f docker-compose.eval.yml up tasker-eval -d --build

tests:
	go test ./... -race