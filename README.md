# Subscription Aggregator

> **Современный микросервисный API для управления подписками с поддержкой платежей, уведомлений и аналитики**

Subscription Aggregator — это полнофункциональная система управления пользовательскими подписками, построенная на микросервисной архитектуре с использованием Go, PostgreSQL, Redis, RabbitMQ и gRPC. Система предоставляет REST API для управления подписками, интегрированную систему платежей через YooKassa, автоматические уведомления и детальную аналитику расходов.

## Основные возможности

### Система аутентификации и авторизации
- JWT-токены для безопасной аутентификации
- gRPC-сервис авторизации для централизованного управления доступом
- Ролевая модель (admin/user) с разграничением прав доступа
- Хеширование паролей с использованием bcrypt
- Автоматическая регистрация администратора при первом запуске

### Управление подписками
- Полный CRUD для подписок (создание, чтение, обновление, удаление)
- Пагинация списков подписок
- Фильтрация и поиск по различным критериям
- Расчет суммарной стоимости подписок за период
- Отслеживание статуса подписок (активные/неактивные)
- Автоматический расчет дат следующих платежей

### Интеграция с платежными системами
- YooKassa API для обработки платежей
- Безопасное хранение токенов карт
- Webhook-обработка для уведомлений о платежах
- История платежей с детализацией
- Автоматическое продление подписок после успешной оплаты

### Система уведомлений
- RabbitMQ для асинхронной обработки сообщений
- Email-уведомления через SMTP (Mail.ru) с поддержкой STARTTLS
- Автоматические напоминания об истечении подписок
- Уведомления о пробном периоде и необходимости оплаты
- Надежная доставка с повторными попытками

### Микросервисная архитектура
- Scheduler — планировщик задач и поиск истекающих подписок
- Sender — сервис отправки уведомлений
- Auth — gRPC-сервис авторизации
- Main API — основной HTTP API сервис

## Технологический стек

### Backend
- Go 1.23.1 — основной язык разработки
- Chi Router — HTTP роутер
- gRPC — межсервисное взаимодействие
- JWT — аутентификация и авторизация
- bcrypt — хеширование паролей

### Базы данных и кеширование
- PostgreSQL 15 — основная база данных
- Redis 7 — кеширование и сессии
- Golang Migrate — управление миграциями

### Инфраструктура
- RabbitMQ 3 — очереди сообщений
- Docker & Docker Compose — контейнеризация
- SMTP (Mail.ru) — отправка email
- YooKassa API — платежная система

### Документация и тестирование
- Swagger/OpenAPI — документация API
- Testcontainers — интеграционные тесты
- Stretchr/testify — unit-тесты
- golangci-lint — статический анализ кода

## Быстрый старт

### Требования
- Docker и Docker Compose
- Go 1.23+ (для локальной разработки)
- Git

### Клонирование репозитория
```bash
git clone https://github.com/your-username/subscription-aggregator.git
cd subscription-aggregator
```

### Настройка окружения
Создайте файл `.env` в корне проекта:
```env
# Database
POSTGRES_DB=subscription_aggregator
POSTGRES_USER=postgres
POSTGRES_PASSWORD=your_password
POSTGRES_PORT=5432
POSTGRES_DATA=./data/postgres

# Redis
REDIS_PASSWORD=your_redis_password
REDIS_PORT=6379
REDIS_DATA=./data/redis

# RabbitMQ
RABBITMQ_PORT_AMQP=5672
RABBITMQ_PORT_MANAGEMENT=15672

# Application
APP_PORT=8080
AUTH_PORT=50051
CONFIG_PATH=./config/config.yaml
CONFIG_PATH_VOLUMES=./config
```

### Запуск через Docker Compose
```bash
# Запуск всех сервисов
docker-compose up -d

# Просмотр логов
docker-compose logs -f

# Остановка сервисов
docker-compose down
```

### Проверка работоспособности
```bash
# Проверка API
curl http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com","password":"password123"}'

# Swagger документация
open http://localhost:8080/docs/
```

## Структура базы данных

![Database Schema](./assets/db.png)

### Основные таблицы:
- **users** — пользователи системы
- **subscriptions** — подписки пользователей
- **payment_tokens** — токены карт для платежей
- **payments** — история платежей

## API Endpoints

### Аутентификация
| Метод | Endpoint | Описание |
|-------|----------|----------|
| `POST` | `/api/v1/register` | Регистрация нового пользователя |
| `POST` | `/api/v1/login` | Авторизация и получение JWT токена |

### Управление подписками
| Метод | Endpoint | Описание |
|-------|----------|----------|
| `POST` | `/api/v1/subscriptions` | Создание новой подписки |
| `GET` | `/api/v1/subscriptions/{id}` | Получение подписки по ID |
| `PUT` | `/api/v1/subscriptions/{id}` | Обновление подписки |
| `DELETE` | `/api/v1/subscriptions/{id}` | Удаление подписки |
| `GET` | `/api/v1/subscriptions/list` | Список подписок с пагинацией |
| `POST` | `/api/v1/subscriptions/sum` | Расчет суммы подписок |

### Платежи
| Метод | Endpoint | Описание |
|-------|----------|----------|
| `POST` | `/api/v1/payment` | Создание платежа |
| `GET` | `/api/v1/payments/list` | История платежей |

## Архитектура системы

```mermaid
graph TB
    Client[Web Client] --> API[Main API Service :8080]
    API --> Auth[Auth gRPC Service :50051]
    API --> DB[(PostgreSQL)]
    API --> Cache[(Redis)]
    API --> Payment[YooKassa API]
    
    Scheduler[Scheduler Service] --> DB
    Scheduler --> RabbitMQ[RabbitMQ]
    
    Sender[Sender Service] --> RabbitMQ
    Sender --> SMTP[SMTP Mail.ru]
    
    RabbitMQ --> Scheduler
    RabbitMQ --> Sender
```

## Конфигурация

Основные настройки находятся в `config/config.yaml`:

```yaml
env: "local"
grpc_auth_address: "auth:50051"
storage_connection_string: "postgres://user:pass@db:5432/db?sslmode=disable"
redis_connection:
  addressredis: "redis:6379"
  password: "your_password"
http_server:
  addresshttp: ":8080"
  timeouthttp: 4s
jwttoken:
  jwt_secret_key: "your-secret-key"
  token_ttl: 24h
smtp:
  smtp_host: smtp.mail.ru
  smtp_port: 587
  smtp_user: "your-email@mail.ru"
  smtp_pass: "your-password"
rabbitmq:
  rabbitmq_url: "amqp://guest:guest@rabbitmq:5672/"
```

## Тестирование

### Запуск тестов
```bash
# Unit тесты
go test ./...

# Интеграционные тесты
go test -tags=integration ./...

# Тесты с покрытием
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Тестирование API
```bash
# Регистрация пользователя
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com", 
    "password": "password123"
  }'

# Авторизация
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "password123"
  }'
```

## Мониторинг и логирование

- **Структурированное логирование** с использованием `slog`
- **Graceful shutdown** для корректного завершения работы
- **Health checks** для всех сервисов
- **Метрики производительности** и обработки ошибок

## Безопасность

- **JWT токены** с настраиваемым временем жизни
- **Хеширование паролей** с bcrypt
- **Валидация входных данных** с детальными сообщениями об ошибках
- **HTTPS поддержка** (в production)
- **Защищенные соединения** для SMTP (STARTTLS)
