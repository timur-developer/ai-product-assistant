# AI Product Assistant

HTTP API сервис для структурирования и доработки продуктовых идей с использованием искусственного интеллекта.

Сервис принимает сырое описание продуктовой идеи и возвращает структурированный подробный черновик: суть, целевая аудитория, ценность, сценарии использования, ограничения, риски и уточняющие вопросы.

## Описание

**Цель**: Помощь продуктологам в быстром структурировании и развитии идей через интеграцию с LLM-провайдерами.

**Основные возможности**:
- Генерация структурированного черновика из сырого описания идеи
- Доработка отдельных секций черновика на основе обратной связи
- Полная история версий черновиков
- Версионирование с отслеживанием использования токенов
- Rate limiting для защиты сервиса
- Graceful shutdown и health-check endpoints

## Архитектура

```
Handler Layer (HTTP boundary)
     ↓
Service/Usecase Layer (Business logic)
     ↓
Repository Layer (Data access)
```

**Ключевые компоненты**:
- **Handler**: HTTP endpoints, валидация входных данных, маппинг DTO
- **Service**: Бизнес-логика генерации и доработки черновиков
- **Repository**: CRUD операции и управление версиями
- **LLM Client**: Интеграция с OpenAI-совместимым API
- **Storage**: PostgreSQL для персистенции данных

## Требования

- Go 1.24+
- PostgreSQL 16+
- Docker и Docker Compose (для локального запуска)
- OpenAI API ключ (или совместимый LLM-провайдер)

## Локальный запуск

### Способ 1: Docker Compose (рекомендуется)

```bash
# 1. Скопировать и настроить переменные окружения
cp .env.example .env

# 2. Добавить ваш OpenAI API ключ в .env
# Отредактируйте .env и установите:
#   LLM_API_KEY=sk-your-actual-key
#   LLM_BASE_URL=https://api.openai.com/v1  (если используете OpenAI)
#   LLM_MODEL=gpt-4

# 3. Запустить stack
docker-compose up -d

# 4. Проверить статус
docker-compose ps
```

Сервис будет доступен на `http://localhost:8080`

### Способ 2: Локальный запуск (требует PostgreSQL)

```bash
# 1. Установить зависимости Go
go mod download

# 2. Создать базу данных PostgreSQL
createdb -U postgres ai_product_assistant

# 3. Применить миграции
psql -U postgres -d ai_product_assistant < migrations/000001_create_drafts.up.sql
psql -U postgres -d ai_product_assistant < migrations/000002_create_draft_versions.up.sql

# 4. Скопировать и настроить переменные окружения
cp .env.example .env

# 5. Запустить приложение
DATABASE_URL="postgres://postgres:postgres@localhost:5432/ai_product_assistant?sslmode=disable" \
LLM_API_KEY="sk-your-actual-key" \
LLM_BASE_URL="https://api.openai.com/v1" \
LLM_MODEL="gpt-4" \
go run ./cmd/app/
```

## Конфигурация

Все параметры настраиваются через переменные окружения. Обязательные:

| Переменная | Описание | Пример |
|-----------|---------|--------|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://user:pass@localhost/db` |
| `LLM_API_KEY` | API ключ для LLM провайдера | `sk-...` |
| `LLM_BASE_URL` | Base URL LLM API | `https://api.openai.com/v1` |
| `LLM_MODEL` | Модель для использования | `gpt-4` |

Опциональные:

| Переменная | Значение по умолчанию | Описание |
|-----------|-------------------|----|
| `APP_ENV` | `dev` | Окружение (dev/prod) |
| `HTTP_PORT` | `8080` | Порт HTTP сервера |
| `LLM_TIMEOUT_SEC` | `15` | Таймаут запроса к LLM |
| `LLM_MAX_RETRIES` | `2` | Максимум попыток переподключения |
| `HTTP_RATE_LIMIT_RPM` | `20` | Ограничение: макс 20 запросов в минуту с IP |
| `HTTP_READ_TIMEOUT_SEC` | `10` | Таймаут чтения HTTP |
| `HTTP_WRITE_TIMEOUT_SEC` | `10` | Таймаут записи HTTP |

## API Endpoints

### Health Check

```bash
GET /health
```

Ответ:
```json
{ "status": "ok" }
```

### Readiness Check

```bash
GET /ready
```

Ответ:
```json
{ "status": "ready" }
```

### Генерирование черновика

```bash
POST /drafts/generate
Content-Type: application/json

{
  "raw_idea": "Мобильное приложение для отслеживания расходов с AI-ассистентом",
  "language": "ru"
}
```

Ответ (201 Created):
```json
{
  "id": 1,
  "raw_idea": "...",
  "language": "ru",
  "latest_version": 1,
  "created_at": "2026-03-26T10:30:00Z",
  "updated_at": "2026-03-26T10:30:00Z",
  "version": {
    "id": 1,
    "draft_id": 1,
    "version": 1,
    "content": {
      "summary": "...",
      "audience": "...",
      "value": "...",
      "scenarios": [...],
      "constraints": [...],
      "risks": [...],
      "questions": [...]
    },
    "provider": "openai",
    "model_name": "gpt-4",
    "usage": {
      "prompt_tokens": 500,
      "completion_tokens": 800,
      "total_tokens": 1300
    },
    "created_at": "2026-03-26T10:30:00Z"
  }
}
```

### Получение черновика

```bash
GET /drafts/{id}
```

Возвращает текущую версию черновика с той же структурой, что и endpoint генерирования.

### Список черновиков

```bash
GET /drafts?limit=10&offset=0
```

Ответ:
```json
{
  "items": [
    {
      "id": 1,
      "raw_idea": "...",
      "language": "ru",
      "latest_version": 1,
      "created_at": "2026-03-26T10:30:00Z",
      "updated_at": "2026-03-26T10:30:00Z"
    }
  ]
}
```

### Доработка черновика

```bash
POST /drafts/{id}/refine
Content-Type: application/json

{
  "sections": ["summary", "value"],
  "language": "ru"
}
```

Создает новую версию черновика с доработанными секциями. Параметры:
- `sections`: Массив секций для доработки (summary, audience, value, scenarios, constraints, risks, questions)
- `language`: Язык для обработки (только ru)

Ответ (200 OK): Черновик с новой версией

## Примеры использования

### cURL

#### Генерирование черновика
```bash
curl -X POST http://localhost:8080/drafts/generate \
  -H "Content-Type: application/json" \
  -d '{
    "raw_idea": "Приложение для управления личным бюджетом с интеграцией банковских счетов",
    "language": "ru"
  }'
```

#### Получение черновика
```bash
curl http://localhost:8080/drafts/1
```

#### Список черновиков
```bash
curl "http://localhost:8080/drafts?limit=10&offset=0"
```

#### Доработка черновика
```bash
curl -X POST http://localhost:8080/drafts/1/refine \
  -H "Content-Type: application/json" \
  -d '{
    "sections": ["summary", "value", "risks"],
    "language": "ru"
  }'
```

### Python

```python
import requests
import json

BASE_URL = "http://localhost:8080"

# Генерирование черновика
response = requests.post(
    f"{BASE_URL}/drafts/generate",
    json={
        "raw_idea": "Приложение для управления личным бюджетом",
        "language": "ru"
    }
)
draft = response.json()
draft_id = draft["id"]
print(f"Создан черновик #{draft_id}")

# Получение черновика
response = requests.get(f"{BASE_URL}/drafts/{draft_id}")
draft = response.json()
print(f"Резюме: {draft['version']['content']['summary']}")

# Доработка черновика
response = requests.post(
    f"{BASE_URL}/drafts/{draft_id}/refine",
    json={
        "sections": ["value", "risks"],
        "language": "ru"
    }
)
updated_draft = response.json()
print(f"Версия обновлена до {updated_draft['latest_version']}")
```

## Коды ошибок

API возвращает структурированные ошибки:

```json
{
  "error": {
    "code": "invalid_request",
    "message": "raw_idea is required"
  }
}
```

| Код HTTP | Код ошибки | Описание |
|---------|-----------|---------|
| 400 | `invalid_request` | Ошибка валидации входных данных |
| 404 | `not_found` | Черновик не найден |
| 409 | `conflict` | Конфликт (напр. черновик был обновлен) |
| 429 | `rate_limited` | Превышено ограничение по количеству запросов |
| 502 | `provider_failed` | Ошибка LLM провайдера |
| 500 | `internal_error` | Внутренняя ошибка сервера |

## Ограничения

- Максимум 4000 символов в поле `raw_idea`
- Максимум 20 запросов в минуту с одного IP адреса
- Поддерживаемый язык: только `ru` (русский)
- Таймаут для запросов к LLM: 15 секунд (по умолчанию)

## Структура проекта

```
.
├── cmd/app/                    # Entry point приложения
│   └── main.go
├── config/                     # Загрузка конфигурации
│   └── config.go
├── internal/
│   ├── handler/
│   │   ├── draft/              # HTTP handlers для drafts
│   │   │   ├── dto.go          # Request/Response DTOs
│   │   │   ├── handler.go      # Handlers
│   │   │   ├── mapper.go       # Mapping logic
│   │   │   └── validation.go   # Request validation
│   │   └── httpapi/
│   │       ├── errors.go       # Error handling
│   │       └── write.go        # Response writing
│   ├── llm/
│   │   └── openai/             # OpenAI LLM client
│   │       ├── client.go
│   │       ├── errors.go
│   │       ├── types.go
│   │       └── prompts.go
│   ├── middleware/
│   │   ├── clientip.go         # Client IP extraction
│   │   └── ratelimit.go        # Rate limiting middleware
│   ├── model/                  # Domain models
│   │   ├── draft.go
│   │   ├── draft_version.go
│   │   ├── llm.go
│   │   └── content.go
│   ├── repository/             # Data access layer
│   │   └── draft_repository.go
│   ├── server/                 # HTTP server setup
│   │   └── server.go
│   ├── storage/
│   │   ├── postgres/           # PostgreSQL driver
│   │   │   └── postgres.go
│   │   └── transaction/        # Transaction management
│   │       ├── context.go
│   │       └── manager.go
│   └── usecase/                # Business logic
│       ├── contracts.go        # Interfaces
│       ├── service.go          # Main service
│       ├── errors.go           # Domain errors
│       └── validation.go       # Business validation
├── migrations/                 # SQL миграции
│   ├── 000001_create_drafts.up.sql
│   ├── 000001_create_drafts.down.sql
│   ├── 000002_create_draft_versions.up.sql
│   └── 000002_create_draft_versions.down.sql
├── docker-compose.yml          # Docker Compose конфиг
├── Dockerfile                  # Application container
├── go.mod                      # Go modules
├── go.sum                      # Go modules checksums
└── README.md                   # Этот файл
```

## Разработка и тестирование

### Запуск тестов

```bash
# Все тесты
go test ./...

# С покрытием
go test -cover ./...

# Интеграционные тесты (требуют БД)
go test -tags=integration ./...
```

### Лучшие практики

- Все логи структурированы через `log/slog`
- Ошибки оборачиваются с контекстом через `%w`
- Бизнес-логика находится в `usecase` слое
- Валидация входных данных происходит в `handler` слое
- Репозитарий абстрагирует детали хранилища

## Troubleshooting

### Ошибка подключения к БД
```
DATABASE_URL is required
```
Убедитесь, что переменная окружения `DATABASE_URL` установлена:
```bash
export DATABASE_URL="postgres://user:pass@host:5432/db"
```

### Ошибка LLM провайдера
```json
{
  "error": {
    "code": "provider_failed",
    "message": "provider failed"
  }
}
```
Проверьте:
- Правильность `LLM_API_KEY`
- Доступность `LLM_BASE_URL`
- Существование модели `LLM_MODEL`
- Лимиты на вашем LLM аккаунте

### Rate limit
```json
{
  "error": {
    "code": "rate_limited",
    "message": "too many requests"
  }
}
```
Превышено ограничение по количеству запросов. По умолчанию 20 запросов в минуту.

## Миграции БД

Миграции применяются автоматически при запуске через Docker Compose.

При локальном запуске примените вручную:
```bash
psql -U postgres -d ai_product_assistant -f migrations/000001_create_drafts.up.sql
psql -U postgres -d ai_product_assistant -f migrations/000002_create_draft_versions.up.sql
```

### Откат миграций
```bash
psql -U postgres -d ai_product_assistant -f migrations/000002_create_draft_versions.down.sql
psql -U postgres -d ai_product_assistant -f migrations/000001_create_drafts.down.sql
```

## Безопасность

- API ключи хранятся только в переменных окружения
- Секреты не логируются
- Внутренние детали ошибок не возвращаются в API
- Применяется rate limiting для защиты от DDoS
- Используется HTTPS в production (настраивается через reverse proxy)

## Performance

- PostgreSQL индексы оптимизируют поиск
- Версионирование позволяет отследить историю изменений
- Кэширование последней версии черновика в памяти
- Graceful shutdown обеспечивает корректное завершение работы

## Лицензия

Приватный проект

## Автор

Разработано как внутренний AI-помощник для продуктологов

## Дальнейшее развитие

- Добавление поддержки других LLM провайдеров
- Экспорт черновиков в Markdown/PDF
- Добавление аутентификации и авторизации
- Улучшение метрик и мониторинга
- Интеграция с внешними хранилищами (S3, Google Drive)
- Реальный взаимодействие между пользователями

