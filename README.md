# LLM вариант 8 — Backend (транспорт + прикладной)

Репозиторий содержит **транспортный** (Go, Kafka) и **прикладной** (Go + Gin, WebSocket + REST) уровни. Агентный уровень — отдельный репозиторий [TESTGENAGENT](../TESTGENAGENT).

## Архитектура

```
Frontend / Postman
    → App :8080 (WS /ws, REST /api/*)
    → Transport :8081 (/v1/messages/*, /v1/segments)
    → Kafka → сборка сегментов (потери R=8%, цикл N с)
    ← Agent :8082 (отдельный процесс)
```

Порты по умолчанию: **8080** (app), **8081** (transport), **9092** (Kafka).

## Быстрый старт (Docker)

```bash
cp .env.example .env
# Запустите агент отдельно (см. TESTGENAGENT/README.md) на :8082
docker compose up --build
```

Проверка:

```bash
curl http://localhost:8081/health
curl http://localhost:8080/health
```

## Локальная разработка

### 1. Kafka + transport

```bash
docker compose up -d zookeeper kafka
cd transport && go mod tidy && go run ./cmd/server
```

Переменные: `KAFKA_BROKERS=localhost:9092`, `AGENT_BASE_URL=http://localhost:8082`, `SEGMENT_LOSS_PERCENT=8`, `ASSEMBLY_INTERVAL_SEC=2`.

### 2. App backend

```bash
cd app && go mod tidy && go run ./cmd/server
```

Переменные: `TRANSPORT_BASE_URL=http://localhost:8081`, `APP_PORT=8080` (или `HTTP_ADDR=:8080`), `RECEIVE_WAIT_MS`, `RECEIVE_MAX_ATTEMPTS`, `RECEIVE_POLL_INTERVAL_MS`.

Структура (`чистая архитектура`):

```
app/
  cmd/server/main.go
  internal/domain/          # интерфейсы TransportClient, ClientHub
  internal/usecase/         # session, generation, chat, proxy
  internal/delivery/http/   # Gin REST
  internal/delivery/ws/     # WebSocket /ws
  internal/infrastructure/  # config, transport HTTP client, hub
```

### 3. Агент (отдельный репозиторий)

```bash
cd ../TESTGENAGENT
python -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt
uvicorn app.main:app --host 0.0.0.0 --port 8082
```

## REST / WebSocket

### Транспорт (`openapi/transport.yaml`)

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/v1/segments` | Сегмент от агента (8% потерь до Kafka) |
| POST | `/v1/messages/send` | Запрос генерации → `message_id`, вызов агента |
| POST | `/v1/messages/receive` | Long-poll, `wait_ms` |

### Прикладной REST (`/api`)

- `POST /api/session` — `{ "email": "user@mail.ru" }` → `{ "sender" }`
- `POST /api/generate` — `{ sender, json_schema, sample_count, constraints? }`
- `POST /api/result` — `{ sender?, wait_ms? }`

Прокси для фронта: `POST /v1/messages/send|receive` на порту **8080**.

### Интеграция с React-фронтендом ([LLM-8-Frontend](https://github.com/Existsq/LLM-8-Frontend))

Текущий фронтенд **не использует WebSocket** — только REST `POST /v1/messages/send` и `POST /v1/messages/receive`.

В `vite.config.ts` по умолчанию прокси ведёт на транспорт `:8081`. Для прохождения через прикладной уровень (сессия, единая точка CORS, будущий WS) укажите:

```ts
// vite.config.ts
server: {
  proxy: {
    '/v1': 'http://localhost:8080',  // app backend, не :8081
    '/api': 'http://localhost:8080',
    '/ws': { target: 'ws://localhost:8080', ws: true },
  },
},
```

Чат в реальном времени (имя, broadcast, logout) — только через `ws://localhost:8080/ws` (см. ниже). Пока фронт на REST, генерация JSON работает через прокси `/v1` на **8080**.

### WebSocket `ws://localhost:8080/ws`

1. `{ "type": "auth", "username": "ivan" }`
2. Генерация: `{ "type": "generate", "json_schema": {...}, "sample_count": 3, "constraints": "..." }`
3. Чат: `{ "type": "send", "payload": "текст" }`
4. Ответ: `{ "type": "receive", "sender", "sent_at", "error", "payload" }`

## Пример curl (сквозной тест)

```bash
curl -s -X POST http://localhost:8081/v1/messages/send \
  -H 'Content-Type: application/json' \
  -d '{
    "sender": "demo",
    "sent_at": "2024-05-18T12:00:00Z",
    "json_schema": {
      "type": "object",
      "properties": {
        "id": { "type": "integer" },
        "name": { "type": "string" }
      }
    },
    "sample_count": 2
  }'

curl -s -X POST http://localhost:8081/v1/messages/receive \
  -H 'Content-Type: application/json' \
  -d '{"sender":"demo","wait_ms":15000}'
```

## Развёртывание (РСА)

| Компонент | Требования | Порт |
|-----------|------------|------|
| Docker | 4 GB RAM, Compose v2 | — |
| transport | Go 1.22 или образ | 8081 |
| app | Go 1.22 или образ | 8080 |
| Kafka+ZK | образы Confluent | 9092, 2181 |
| agent | **отдельная машина/репо** | 8082 |

Для демонстрации на трёх ПК в одной LAN используйте [ZeroTier](https://www.zerotier.com/): укажите IP ноутбука с агентом в `AGENT_BASE_URL` (например `http://192.168.192.10:8082`).

## Ограничения

- Потеря 8% сегментов может привести к `error=true` после 3 циклов сборки без полного набора.
- История чата не персистится.
- Агент требует Ollama (`OLLAMA_MODEL=mistral:7b`, `ollama pull mistral:7b`); при сбое — `error=true` на прикладном, без mock (кроме `ALLOW_MOCK=true` на агенте).
