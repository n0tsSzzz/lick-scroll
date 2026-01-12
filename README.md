# Лизни и Пролистни (Lick Scroll)

MVP для инновационной 18+ платформы - гибрид TikTok и OnlyFans.

## Описание

Платформа для публикации и монетизации контента 18+ от реальных креаторов и AI-генераций. Пользователи могут просматривать контент, подписываться на креаторов, покупать отдельные посты за внутреннюю валюту.

## Архитектура

Проект построен на микросервисной архитектуре с использованием Go:

### Сервисы

1. **Auth Service** (порт 8001) - Аутентификация и авторизация пользователей
2. **Post Service** (порт 8002) - Управление постами, загрузка контента в S3
3. **Feed Service** (порт 8003) - Составление ленты контента из кэша
4. **Fanout Service** (порт 8004) - Добавление постов в ленты подписчиков
5. **Wallet Service** (порт 8005) - Управление внутренней валютой и покупками
6. **Notification Service** (порт 8006) - Отправка уведомлений пользователям
7. **Moderation Service** (порт 8007) - Модерация контента
8. **Analytics Service** (порт 8008) - Аналитика для креаторов

### Инфраструктура

- **PostgreSQL** - основная база данных
- **Redis** - кэширование и очереди
- **AWS S3** - хранение медиафайлов

## Требования

- Go 1.21+
- Docker и Docker Compose
- AWS аккаунт с настроенным S3 bucket (для production)

## Установка и запуск

### 1. Клонирование репозитория

```bash
git clone <repository-url>
cd lick-scroll
```

### 2. Настройка переменных окружения

Создайте файл `.env` в корне проекта:

```env
# JWT Secret (обязательно измените в production!)
JWT_SECRET=your-super-secret-jwt-key-change-in-production

# AWS S3 (для загрузки медиа)
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your-access-key
AWS_SECRET_ACCESS_KEY=your-secret-key
S3_BUCKET_NAME=lick-scroll-content
```

### 3. Запуск через Docker Compose

```bash
docker-compose up -d
```

Это запустит все сервисы и зависимости (PostgreSQL, Redis).

### 4. Проверка работоспособности

Проверьте health check каждого сервиса:

```bash
curl http://localhost:8001/health  # Auth
curl http://localhost:8002/health  # Post
curl http://localhost:8003/health  # Feed
curl http://localhost:8004/health  # Fanout
curl http://localhost:8005/health  # Wallet
curl http://localhost:8006/health  # Notification
curl http://localhost:8007/health  # Moderation
curl http://localhost:8008/health  # Analytics
```

## Swagger Documentation

Каждый сервис имеет Swagger/OpenAPI документацию, доступную через веб-интерфейс:

- **Auth Service**: http://localhost:8001/swagger/index.html
- **Post Service**: http://localhost:8002/swagger/index.html
- **Feed Service**: http://localhost:8003/swagger/index.html
- **Wallet Service**: http://localhost:8005/swagger/index.html
- **Analytics Service**: http://localhost:8008/swagger/index.html

Для генерации документации используйте скрипт:
```bash
./scripts/generate-swagger.sh
```

Или вручную для каждого сервиса:
```bash
cd services/<service-name>
swag init -g main.go --output docs
```

## API Endpoints

### Auth Service (8001)

- `POST /api/v1/register` - Регистрация пользователя
- `POST /api/v1/login` - Вход в систему
- `GET /api/v1/me` - Получить информацию о текущем пользователе

### Post Service (8002)

- `POST /api/v1/posts` - Создать пост (только для креаторов)
- `GET /api/v1/posts/:id` - Получить пост
- `GET /api/v1/posts` - Список постов
- `PUT /api/v1/posts/:id` - Обновить пост
- `DELETE /api/v1/posts/:id` - Удалить пост
- `GET /api/v1/posts/creator/:creator_id` - Посты креатора

### Feed Service (8003)

- `GET /api/v1/feed` - Получить ленту пользователя
- `GET /api/v1/feed/category/:category` - Лента по категории

### Fanout Service (8004)

- `POST /api/v1/fanout/post/:post_id` - Распределить пост по лентам подписчиков
- `POST /api/v1/subscribe/:creator_id` - Подписаться на креатора
- `DELETE /api/v1/subscribe/:creator_id` - Отписаться от креатора

### Wallet Service (8005)

- `GET /api/v1/wallet` - Получить баланс кошелька
- `POST /api/v1/wallet/topup` - Пополнить баланс
- `POST /api/v1/wallet/purchase/:post_id` - Купить пост
- `GET /api/v1/wallet/transactions` - История транзакций

### Notification Service (8006)

- `POST /api/v1/notifications/send` - Отправить уведомление пользователю
- `POST /api/v1/notifications/broadcast` - Массовая рассылка уведомлений

### Moderation Service (8007)

- `POST /api/v1/moderation/review/:post_id` - Проверить пост
- `GET /api/v1/moderation/pending` - Список постов на модерации
- `POST /api/v1/moderation/approve/:post_id` - Одобрить пост
- `POST /api/v1/moderation/reject/:post_id` - Отклонить пост

### Analytics Service (8008)

- `GET /api/v1/analytics/creator/stats` - Статистика креатора
- `GET /api/v1/analytics/creator/posts/:post_id` - Статистика поста
- `GET /api/v1/analytics/creator/revenue` - Доходы креатора

## Примеры использования

### Регистрация пользователя

```bash
curl -X POST http://localhost:8001/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "username": "username",
    "password": "password123",
    "role": "viewer"
  }'
```

### Создание поста (креатор)

```bash
curl -X POST http://localhost:8002/api/v1/posts \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "title=My Post" \
  -F "description=Post description" \
  -F "type=photo" \
  -F "category=fetish" \
  -F "price=100" \
  -F "media=@/path/to/image.jpg"
```

### Получение ленты

```bash
curl -X GET http://localhost:8003/api/v1/feed \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

## Структура проекта

```
lick-scroll/
├── pkg/                    # Общие пакеты
│   ├── config/            # Конфигурация
│   ├── database/          # Подключение к БД
│   ├── cache/             # Redis клиент
│   ├── jwt/               # JWT сервис
│   ├── middleware/        # HTTP middleware
│   ├── models/            # Модели данных
│   ├── s3/                # S3 клиент
│   └── logger/            # Логирование
├── services/              # Микросервисы
│   ├── auth/
│   ├── post/
│   ├── feed/
│   ├── fanout/
│   ├── wallet/
│   ├── notification/
│   ├── moderation/
│   └── analytics/
├── docker-compose.yml     # Docker Compose конфигурация
└── README.md
```

## Бизнес-логика

### Роли пользователей

- **viewer** - зритель, может просматривать и покупать контент
- **creator** - креатор, может создавать и монетизировать контент
- **moderator** - модератор, проверяет контент перед публикацией

### Процесс публикации поста

1. Креатор загружает контент через Post Service
2. Контент сохраняется в S3
3. Пост создается со статусом "pending"
4. Moderation Service проверяет контент
5. После одобрения Fanout Service добавляет пост в ленты подписчиков
6. Notification Service уведомляет подписчиков о новом контенте

### Покупка поста

1. Пользователь запрашивает покупку через Wallet Service
2. Проверяется баланс кошелька
3. Если достаточно средств, баланс списывается
4. Создается транзакция
5. Пользователь получает доступ к контенту

## Разработка

### Локальная разработка

Для разработки без Docker:

```bash
# Установка зависимостей
go mod download

# Запуск PostgreSQL и Redis (через Docker)
docker-compose up -d postgres redis

# Запуск сервиса (например, auth)
cd services/auth
go run main.go
```

### Тестирование

```bash
# Запуск всех тестов
go test ./...

# Тесты конкретного сервиса
cd services/auth
go test ./...
```

## Безопасность

- Все API endpoints (кроме регистрации и входа) требуют JWT токен
- Пароли хешируются с помощью bcrypt
- Rate limiting для защиты от злоупотреблений
- Валидация всех входных данных

## Масштабирование

- Каждый сервис может масштабироваться независимо
- Redis используется для кэширования и снижения нагрузки на БД
- S3 для хранения медиафайлов обеспечивает масштабируемость

## Лицензия

[Укажите лицензию]

## Контакты

[Укажите контакты]

