# Web Behavior Backend

### Кратко
Бэкенд-сервис для сбора, хранения и аналитики пользовательского поведения в вебе. Реализованы сбор событий, агрегирующие метрики, AI‑аналитика, управление пользователями и организациями, а также интеграция с Chrome‑расширением.

- **Стек**: Go (Gin, sqlx), PostgreSQL, Redis, JWT, Swagger
- **Артефакты**: REST API (`/api/v1`), Swagger UI (`/swagger`)
- **Хранилища**: PostgreSQL, Redis
- **Деплой**: Docker/Docker Compose, манифесты `k3s/`

---

## Требования
- Go 1.24+
- Docker 24+ и Docker Compose 2+
- (Для локальной разработки) PostgreSQL 15, Redis 7 — могут подниматься через `docker-compose.local.yml`

---

## Быстрый старт (локально)
1) Подготовьте `.env` в корне (см. пример ниже)
2) Поднимите инфраструктуру (БД и Redis):
```bash
docker compose -f docker-compose.local.yml up -d
```
3) Примените миграции:
```bash
make migrate
```
4) Запустите сервер:
```bash
make serve
```
Сервис поднимется на `http://localhost:8080`. Проверка здоровья: `GET /health`.

Swagger UI: `http://localhost:8080/swagger/index.html`

---

## Переменные окружения (.env пример)
```dotenv
# Общие
ENV=dev
PORT=8080
BASE_URL=http://localhost:8080
JWT_SECRET=changeme

# База данных
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASS=postgres
DB_NAME=web_behavior
DB_SSLMODE=disable

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=your_redis_password
```
Примечания:
- В Docker окружении `DB_HOST` для backend указывается как имя сервиса БД из compose: `web_behavior_db`.
- В `config/config.go` предусмотрены дефолты, но для продиспользования задавайте значения явно.

---

## Команды Makefile (основные)
```bash
make serve        # Запуск сервера (go run cmd/main.go serve)
make migrate      # Применить миграции (go run cmd/main.go migrate)
make migrate-down # Откатить миграции (go run cmd/main.go migrate -d)
make build        # Сборка бинаря в bin/web-behavior
make test         # Запуск тестов
make clean        # Очистка артефактов сборки
```

---

## Архитектура и директории
- `cmd/` — точка входа и CLI (команды `serve`, `migrate`)
- `server/` — инициализация HTTP‑сервера и роутинг (Gin)
- `config/` — загрузка конфигурации/ENV
- `internal/`:
  - `entity/` — доменные сущности и DTO
  - `handler/` — HTTP‑обработчики (user, user_behavior, metrics, ai-analytics, organization, extension)
  - `service/` — бизнес‑логика, интеграции (Redis, AI‑аналитика и т. д.)
  - `repository/` — доступ к БД (sqlx)
- `middleware/` — аутентификация, авторизация, API‑ключи и пр.
- `migrations/` — SQL миграции (golang-migrate совместимая структура)
- `docs/` — Swagger спецификация (`swagger.yaml`, `swagger.json`)
- `k3s/` — манифесты для деплоя в Kubernetes (k3s)

Высокоуровневый поток:
- Клиент/расширение шлет события в публичные эндпоинты `/api/v1/inayla/*`
- Админ‑панель/приложение работает через приватные админ‑маршруты `/api/v1/admin/*` (JWT)
- Метрики и AI‑аналитика доступны на соответствующих маршрутах в админ‑группе

---

## API
Swagger UI доступен по адресу:
- Локально: `http://localhost:8080/swagger/index.html`

Основные группы маршрутов (см. `server/server.go` и Swagger):
- Публичные для сбора событий:
  - `POST /api/v1/inayla/behaviors`
  - `POST /api/v1/inayla/behaviors/batch`
  - `GET /api/v1/inayla/extension/users/auth` (с `API-Key`, middleware)
- Админ‑аутентификация:
  - `POST /api/v1/admin/users/auth` (логин по паролю, выдает JWT)
- Приватные (JWT): пользователи, организации, метрики, аналитика, управление ключами расширения
- Служебные:
  - `GET /health` — статус сервиса

Актуальные схемы запросов/ответов, коды ошибок — в Swagger (`docs/swagger.yaml`).

---

## Миграции
Миграции располагаются в `migrations/` и применяются командой:
```bash
make migrate
```
Откатить последнюю/все в рамках конфигурации:
```bash
make migrate-down
```

Миграции использует CLI на базе `github.com/spf13/cobra` (см. `cmd/root` и `cmd/migrate`).

---

## Запуск в Docker
Сборка и запуск контейнера приложения:
```bash
docker build -t web-behavior:local .
docker run --env-file .env -p 8080:8080 web-behavior:local
```

Локальная инфраструктура (PostgreSQL, Redis):
```bash
docker compose -f docker-compose.local.yml up -d
```

Прод/стейдж с использованием готового образа:
```bash
docker compose -f docker-compose.yml up -d
```
Обратите внимание на переменные окружения в compose и тома для БД.

---

## Деплой в Kubernetes (k3s)
В папке `k3s/` находятся манифесты: `deployment`, `service`, `ingress`, `configmap`, `secret`, `postgres`, `redis`. Перед применением:
- Проверьте namespace и секреты (JWT, доступы к БД/Redis)
- Синхронизируйте `ConfigMap`/`Secret` с вашим `.env`
- Примените `kubectl apply -f k3s/`

---

## диагностика
- Health check: `GET /health`
- Swagger:  `/swagger/index.html`
- Частые проблемы:
  - Нет соединения с БД — проверьте `DB_HOST`, `DB_USER/DB_PASS`, доступность контейнера `web_behavior_db`
  - Redis не указан/не доступен — задайте `REDIS_*` в `.env` или поднимите Redis из `docker-compose.local.yml`
  - JWT ошибки — проверьте `JWT_SECRET`

