## Air tickets: учет рейсов и продажи

Стек:
- Backend: Go + Gin + pgx
- Frontend: React (Vite)
- DB: PostgreSQL 17 (только в Docker)
- Auth: JWT + OAuth 2.0 (Google)

### Быстрый старт (Docker)

```bash
docker compose up --build
```

Откроется:
- Frontend: `http://localhost:5173`
- Backend: `http://localhost:8080`
- Healthcheck: `http://localhost:8080/healthz`

### OAuth (опционально)

Если хочешь включить вход через Google OAuth2, задай переменные окружения перед стартом:

```bash
export OAUTH_GOOGLE_CLIENT_ID="..."
export OAUTH_GOOGLE_CLIENT_SECRET="..."
export OAUTH_GOOGLE_REDIRECT_URL="http://localhost:8080/api/auth/oauth/google/callback"
docker compose up --build
```

### Demo-логин

После старта можно зарегистрировать пользователя:

`POST /api/auth/register` `{ "email": "a@b.c", "password": "pass" }`

и войти:

`POST /api/auth/login` `{ "email": "a@b.c", "password": "pass" }`

Ответ вернет `access_token` и `refresh_token`.

`access_token` нужно передавать как `Authorization: Bearer <access_token>`.

Для обновления:

`POST /api/auth/refresh` `{ "refresh_token": "..." }`

