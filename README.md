# Real Time Chat App

Full-stack real-time chat app with a Go Gin backend, React/Vite frontend, Postgres, Redis, JWT auth, and WebSocket messaging.

## Stack

- Backend: Go 1.26.5, Gin, pgx, JWT, bcrypt, Gorilla WebSocket
- Frontend: React, Vite, TypeScript, npm
- Data: Postgres, Redis
- Dev runtime: Docker Compose

## Run With Docker

Start Docker Desktop first, then run:

```sh
docker compose up --build
```

Services:

- Frontend: http://localhost:5173
- Backend: http://localhost:8080
- Postgres: localhost:5432
- Redis: localhost:6379

The `migrate` service runs database migrations before the backend starts.

## Run Locally

Start Postgres and Redis, then create `backend/.env` from `backend/.env.example`:

```sh
cp backend/.env.example backend/.env
```

Apply migrations:

```sh
cd backend
go run ./cmd/migrate
```

Start the backend:

```sh
go run ./cmd/server
```

Start the frontend:

```sh
cd frontend
npm install
npm run dev
```

## Backend Environment

```env
PORT=8080
GIN_MODE=debug
DATABASE_URL=postgres://postgres:postgres@localhost:5432/real_time_chat?sslmode=disable
REDIS_URL=redis://localhost:6379/0
JWT_SECRET=change-me
CORS_ALLOWED_ORIGIN=http://localhost:5173
```

## Useful Checks

Backend:

```sh
cd backend
go test ./...
```

Frontend:

```sh
cd frontend
npm run lint
npm run build
```

## Features

- Register and login with JWT auth
- Conversation list and search
- Direct conversations and rooms
- Room rename, delete, leave, add/remove participants
- Send, edit, and delete messages
- Real-time message updates over WebSocket
- Typing indicators
- Redis-backed rate limiting
