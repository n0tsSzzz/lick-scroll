# Lick Scroll Frontend

Полноценный фронтенд для платформы Lick Scroll, построенный на React + Vite.

## Технологии

- **React 18** - UI библиотека
- **React Router** - Роутинг
- **Vite** - Сборщик
- **Axios** - HTTP клиент

## Установка

```bash
cd frontend/app
npm install
```

## Запуск

```bash
npm run dev
```

Приложение будет доступно по адресу: `http://localhost:3000`

## Сборка

```bash
npm run build
```

## Структура проекта

```
src/
  components/     # Переиспользуемые компоненты
  pages/         # Страницы приложения
  services/      # API клиенты
  styles/        # Глобальные стили
  utils/         # Утилиты
```

## API Endpoints

Фронтенд использует следующие сервисы:
- Auth Service: `http://localhost:8001/api/v1`
- Post Service: `http://localhost:8002/api/v1`
- Feed Service: `http://localhost:8003/api/v1`
- Fanout Service: `http://localhost:8004/api/v1`
- Wallet Service: `http://localhost:8005/api/v1`
- Notification Service: `http://localhost:8006/api/v1`
- Analytics Service: `http://localhost:8008/api/v1`

## Страницы

- `/login` - Вход
- `/register` - Регистрация
- `/` - Лента
- `/profile` - Профиль
- `/create-post` - Создание поста
- `/analytics` - Аналитика
