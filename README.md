# Personage

Веб-приложение «персональный ассистент»: умное управление задачами, расписание и push-уведомления. Бэкенд — микросервисы в Yandex Cloud, фронтенд — React PWA.

## Структура репозитория

```
personage/
├── backend/
│   ├── tasker/            # Go    — задачи, кластеризация событий, LLM-генерация задач, расписание
│   ├── notificator/       # Go    — web-push уведомления (VAPID), SQS-консьюмер
│   ├── Personage.Auth/    # C#    — аутентификация, выдача JWT
│   ├── traitex/           # Py    — извлечение признаков из писем/сообщений (NLP)
│   ├── telegram-auth/     # Py    — менеджер Telegram MTProto-сессий пользователей
│   └── libs/              # общий код: proto, Go-библиотеки (webapp, postgres, sqs, token, …)
├── frontend/              # React 19 + Vite + Tailwind PWA
├── tests/                 # E2E / API-тесты (TS + Jest) для tasker и notificator
└── terraform/             # инфраструктура как код для Yandex Cloud
```

### Сервисы

| Сервис | Стек | Назначение |
|---|---|---|
| `tasker` | Go, PostgreSQL+pgvector, OpenAI/OpenRouter | Принимает события из SQS, кластеризует по embedding-схожести, через LLM решает actionability и генерирует задачи, ведёт расписание |
| `notificator` | Go, PostgreSQL | Хранит подписки и настройки уведомлений, шлёт web-push по сообщениям из SQS |
| `Personage.Auth` | C# / .NET | Регистрация, логин, refresh, JWT для остальных сервисов |
| `traitex` | Python | Обогащает входящие события (Gmail/Telegram) фичами и кладёт в SQS для tasker |
| `telegram-auth` | Python | Серверные MTProto-сессии пользователей Telegram |

### Инфраструктура

- **PostgreSQL 18 + pgvector** — отдельная БД на сервис (`tasker`, `notificator`, `auth`, `traitex`).
- **SQS (Yandex Message Queue)** — асинхронные сообщения между сервисами.
- **gRPC** — синхронные межсервисные вызовы; HTTP-эндпоинты для фронта генерируются grpc-gateway.
- **Миграции** — goose (SQL-файлы в `backend/<service>/migrations/`).

## Запуск

### Требования

- Docker + docker compose
- Go 1.22+, Node 20+, .NET 8, Python 3.11+ (для разработки соответствующих сервисов)
- `yc` CLI с авторизованным аккаунтом (нужен для получения секретов из Yandex Cloud)

### 1. Секреты

`docker compose` подтягивает `../secrets.env`. Сгенерировать:

```bash
make secrets         # из корня репозитория
```

Команда дёргает `yc` и Lockbox и кладёт `YC_TOKEN`, `AWS_*` и т.п. в `secrets.env`.

### 2. Кодогенерация (один раз и после правок `.proto`)

```bash
cd backend
make deps            # установка goose, buf, protoc-плагинов, mockgen, linters
make generate        # buf generate для tasker / notificator / auth
```

> Не запускайте `buf` или `protoc` напрямую — всегда через `make generate` (или `make <service>/generate`).

### 3. Поднять всё локально

```bash
cd backend
make services/deploy     # docker compose up: db + auth + tasker + notificator
```

Порты:
- `5432` — PostgreSQL
- `8080` — HTTP-gateway tasker/notificator (мапятся как `8080`/`8080` соответственно; tasker занят первым — для notificator переопределите port mapping или поднимайте по одному)
- `8082` — Auth REST, `50051` — Auth gRPC

Поднять одно по отдельности:

```bash
make tasker/run
make notificator/run
make tasker-eval/run     # tasker в eval-конфигурации, порты 8091/9091
```

### 4. Миграции

```bash
make tasker/migrate/up
make notificator/migrate/up
make auth/migrate/up
make traitex/migrate/up

# создать новую
make tasker/migrate/create name=add_something

# откат
make tasker/migrate/down          # все
make tasker/migrate/down-one      # одна
```

### 5. Фронтенд

```bash
cd frontend
npm install
npm run dev          # vite dev server
npm run build        # прод-сборка в dist/
npm run typecheck
npm run lint
npm run test:e2e     # Playwright
```

Деплой статики в S3:

```bash
make frontend-release    # build + deploy, из корня; требует secrets.env
```

## Разработка

### Линт / тесты

```bash
cd backend
make lint                # golangci-lint
make tests               # go test ./... -race
make tasker/check-coverage
make notificator/check-coverage
```

### E2E / API-тесты

```bash
cd tests
npm test                 # все
npm run test:tasker
npm run test:watch
```

Тесты используют собственный `docker-compose.yml` в `tests/`.

### Полезное

```bash
make tasker/create-test-tasks   # засеять тестовые задачи в БД tasker
```

## Архитектурные документы

Перед нетривиальными изменениями в потоке «событие → задача» смотреть в `backend/tasker/`:
`FUNCTIONAL_REQUIREMENTS.md`, `USE_CASES.md`, `USE_CASES.puml`, `ARCHITECTURE.puml`.

## CI/CD

GitHub Actions, по воркфлоу на сервис: `*-ci.yml` (codegen → lint → test → build на PR/push) и `*-release.yml` (+ docker push в Yandex Container Registry, миграции БД, terraform deploy). Общие шаблоны — `go-service-ci.yml`, `go-service-release.yml`.
