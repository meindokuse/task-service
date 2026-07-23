# Task Service

REST API для управления задачами в командах с ролевой моделью (RBAC), историей
изменений и сложными SQL-отчётами — реализован в рамках ТЗ.

## Стек

- **Go 1.25**, **Fiber v2** (HTTP)
- **MySQL 8** через `sqlx` (сырой SQL — нужен для JOIN/оконных функций/CTE ниже)
- **Redis** (кэш списка задач)
- **golang-migrate** (миграции схемы, встроены в бинарь и применяются при старте)
- **JWT** (`golang-jwt/jwt/v5`) для аутентификации, **bcrypt** для хэширования паролей
- **sony/gobreaker** — circuit breaker вокруг мок-шлюза email
- **prometheus/client_golang** — метрики на `/metrics`
- **testcontainers-go** — для интеграционных тестов MySQL/Redis

## Архитектура

Clean Architecture / DDD, зависимости направлены внутрь:

```
internal/
├── app.go, init.go        # DI-сборка + жизненный цикл (порядок: config → storage → gateways → services → server)
├── app/{domain}/v1/        # Fiber-хендлеры — только DTO<->транспорт, без бизнес-логики
├── app/middleware/         # JWT-аутентификация, rate limiting, обработка ошибок
├── application/service/    # use case'ы: интерфейсы для своих зависимостей + Execute(ctx, dto)
│   └── {domain}/{use_case}/service.go, registry.go
├── domain/                 # entity + valueobject — чистый Go, без зависимостей от фреймворков
├── infrastructure/
│   ├── storage/{entity}/   # репозитории MySQL (sqlx), маппинг DAO<->entity
│   ├── storage/cache/      # кэш списка задач в Redis
│   └── gateway/email/      # мок email-провайдера + circuit breaker
└── pkg/                    # closer, connector, terror, jwtutil, metrics, txmanager
```

Интерфейсы репозиториев объявляются в use case, который их использует
(`application/service/...`), а реализуются конкретными типами storage/gateway —
инверсия зависимостей, никаких интерфейсов в `infrastructure/`.

## Запуск

```bash
docker compose up --build
```

Поднимает MySQL, Redis и приложение на `:8080`. Миграции применяются
автоматически при старте (`internal/init.go` запускает `golang-migrate` по
встроенным файлам `migrations/*.sql`).

Без Docker: скопируйте `.env.example` в `.env` (либо укажите `CONFIG_PATH` на
YAML-файл — см. `config/config.yaml`) и запустите `go run ./cmd` против своих
MySQL/Redis.

Health check: `GET /healthz`. Метрики: `GET /metrics`.

## API

Все маршруты находятся под `/api/v1`. Все, кроме register/login, требуют
заголовок `Authorization: Bearer <jwt>` (возвращается `/login`).

| Метод | Путь | Примечание |
|---|---|---|
| POST | `/register` | |
| POST | `/login` | |
| POST | `/teams` | вызывающий становится owner |
| GET | `/teams` | команды, в которых состоит вызывающий |
| POST | `/teams/{id}/invite` | только owner/admin; `{"email","role"}` |
| POST | `/tasks` | только участники команды; `{"team_id","title","description","assignee_id"}` |
| GET | `/tasks?team_id=&status=&assignee_id=&page=&page_size=` | кэшируется в Redis, TTL 5 мин |
| PUT | `/tasks/{id}` | creator/assignee/owner-admin команды; частичное обновление, изменения пишутся в историю |
| GET | `/tasks/{id}/history` | |
| POST\|GET | `/tasks/{id}/comments` | |
| GET | `/teams/stats` | сложный запрос (a) |
| GET | `/teams/{id}/top-creators` | сложный запрос (b) |
| GET | `/teams/{id}/orphaned-assignees` | сложный запрос (c), только owner/admin |

ТЗ называет три обязательных сложных запроса, но не задаёт для них маршруты,
поэтому они вынесены в read-эндпоинты (см. `internal/infrastructure/storage/team/repository.go`
для (a) и `internal/infrastructure/storage/task/repository.go` для (b)/(c)):

- **(a)** `teams.Stats` — JOIN 3 таблиц (`teams`+`team_members`+`tasks`) +
  агрегация `COUNT(DISTINCT ...)`: количество участников и количество
  завершённых за 7 дней задач по каждой команде.
- **(b)** `tasks.TopCreators` — CTE + `ROW_NUMBER() OVER (PARTITION BY team_id ...)`:
  топ-3 автора задач за текущий месяц, по каждой команде.
- **(c)** `tasks.OrphanedAssignees` — коррелированный подзапрос `NOT EXISTS`:
  задачи, чей исполнитель не является участником команды задачи (проверка
  целостности данных). Намеренно **не** проверяется при записи (см. комментарий
  в `create_task/service.go`), чтобы этому запросу было что реально находить.

## Тестирование

```bash
go test ./...                                  # unit-тесты (use case'ы, ручные моки)
go test -tags=integration -p 1 ./...           # + интеграционные тесты (нужен Docker)
```

`-p 1` запускает тестовые пакеты последовательно — reaper testcontainers (ryuk)
может конфликтовать, если много контейнеров MySQL стартуют одновременно на
Docker-хосте с ограниченными ресурсами.

Unit-тесты покрывают use case'ы в `application/service/**` (≥85% на пакет) с
написанными вручную моками для собственных интерфейсов репозиториев/шлюзов
каждого use case. Интеграционные тесты поднимают настоящие MySQL 8 (и Redis —
для кэша) через testcontainers-go, применяют реальные миграции и напрямую
проверяют слой хранения — включая три сложных запроса выше.

## Важные архитектурные решения

- **Транзакции через context**: `internal/pkg/txmanager` прикрепляет активный
  `*sqlx.Tx` к `ctx`; репозитории достают его оттуда (`txmanager.Ext(ctx, db)`)
  вместо явного параметра tx, благодаря чему `create_team` (команда + членство
  owner) и `update_task` (обновление задачи + строки истории) остаются
  атомарными, а репозитории при этом не зависят друг от друга.
- **Circuit breaker**: `infrastructure/gateway/email` оборачивает мок
  email-клиента со случайными сбоями в `gobreaker`. Неудачная отправка или
  разомкнутая цепь не приводят к отказу всего запроса приглашения — членство
  всё равно добавляется, а в ответе возвращается `email_delivered: false`.
- **Rate limiting**: 100 запросов/мин, ключом служит ID пользователя из JWT на
  аутентифицированных маршрутах и IP — на `/register`+`/login`.
