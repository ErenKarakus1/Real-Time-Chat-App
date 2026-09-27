# Real Time Chat App

![Go](https://img.shields.io/badge/Go-1.26.5-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![React](https://img.shields.io/badge/React-19-61DAFB?style=for-the-badge&logo=react&logoColor=black)
![TypeScript](https://img.shields.io/badge/TypeScript-6-3178C6?style=for-the-badge&logo=typescript&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-17-4169E1?style=for-the-badge&logo=postgresql&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-8-FF4438?style=for-the-badge&logo=redis&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=for-the-badge&logo=docker&logoColor=white)

A full-stack real-time chat application built with **Go Gin**, **React**, **PostgreSQL**, **Redis**, and **WebSockets**. It supports direct messages, rooms, role-based room management, typing indicators, read receipts, presence, message history pagination, and Docker-based local development.

## Tech Stack

### Backend
- Go 1.26.5
- Gin
- pgx
- PostgreSQL
- Redis
- JWT authentication
- bcrypt password hashing
- Gorilla WebSocket

### Frontend
- React
- Vite
- TypeScript
- npm
- Plain CSS

### DevOps
- Docker
- Docker Compose

## Features

- User registration and login
- JWT-based authentication
- Current user session restore
- User search by username or email
- Online/offline presence
- Direct message conversations
- Room conversations
- Room rename and delete
- Room leave flow
- Add and remove room participants
- Promote member to admin
- Demote admin to member
- Transfer room ownership
- Role-based room controls in the UI
- Send, edit, and delete messages
- Real-time message create/update/delete events
- Real-time read receipts
- Read/sent message state
- Typing indicators
- Conversation unread badges
- Message history pagination on scroll
- Go-to-latest-message shortcut
- Redis-backed rate limiting
- Dockerized local environment

## Project Structure

```text
.
|-- backend
|   |-- cmd
|   |   |-- migrate
|   |   `-- server
|   |-- internal
|   |   |-- auth
|   |   |-- config
|   |   |-- db
|   |   |-- handlers
|   |   |-- httpserver
|   |   |-- middleware
|   |   |-- migrations
|   |   |-- models
|   |   |-- presence
|   |   |-- ratelimit
|   |   |-- realtime
|   |   |-- repositories
|   |   `-- services
|   `-- migrations
|-- frontend
|   |-- public
|   `-- src
`-- docker-compose.yml
```

## Getting Started

### Run With Docker

Start Docker Desktop, then run:

```sh
docker compose up --build
```

Services:

| Service | URL |
| --- | --- |
| Frontend | http://localhost:5173 |
| Backend | http://localhost:8080 |
| PostgreSQL | localhost:5432 |
| Redis | localhost:6379 |

The `migrate` service runs database migrations before the backend starts.

## Run Locally

Start PostgreSQL and Redis first.

Create a backend environment file:

```sh
cp backend/.env.example backend/.env
```

Run migrations:

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

## Environment Variables

Example backend environment:

```env
PORT=8080
GIN_MODE=debug
DATABASE_URL=postgres://postgres:postgres@localhost:5432/real_time_chat?sslmode=disable
REDIS_URL=redis://localhost:6379/0
JWT_SECRET=change-me
CORS_ALLOWED_ORIGIN=http://localhost:5173,http://127.0.0.1:5173
```

## API Overview

### Auth

| Method | Endpoint | Description |
| --- | --- | --- |
| POST | `/auth/register` | Register a user |
| POST | `/auth/login` | Login and receive JWT |
| GET | `/auth/me` | Get current user |

### Users

| Method | Endpoint | Description |
| --- | --- | --- |
| GET | `/users/search` | Search users |
| GET | `/users/presence` | Get online/offline status |

### Conversations

| Method | Endpoint | Description |
| --- | --- | --- |
| GET | `/conversations` | List conversations |
| POST | `/conversations/rooms` | Create room |
| POST | `/conversations/direct` | Create direct conversation |
| PATCH | `/conversations/:conversation_id` | Rename room |
| DELETE | `/conversations/:conversation_id` | Delete room |
| POST | `/conversations/:conversation_id/read` | Mark conversation read |
| PATCH | `/conversations/:conversation_id/owner` | Transfer ownership |

### Participants

| Method | Endpoint | Description |
| --- | --- | --- |
| GET | `/conversations/:conversation_id/participants` | List participants |
| POST | `/conversations/:conversation_id/participants` | Add participant |
| PATCH | `/conversations/:conversation_id/participants/:user_id/role` | Update role |
| DELETE | `/conversations/:conversation_id/participants/me` | Leave room |
| DELETE | `/conversations/:conversation_id/participants/:user_id` | Remove participant |

### Messages

| Method | Endpoint | Description |
| --- | --- | --- |
| GET | `/conversations/:conversation_id/messages` | List messages |
| POST | `/conversations/:conversation_id/messages` | Create message |
| PATCH | `/conversations/:conversation_id/messages/:message_id` | Edit message |
| DELETE | `/conversations/:conversation_id/messages/:message_id` | Delete message |

Message listing supports pagination:

```text
GET /conversations/:conversation_id/messages?before=2026-09-27T10:00:00Z&limit=30
```

### WebSocket

```text
GET /ws/conversations/:conversation_id?token=<jwt>
```

Realtime events:

```text
message.created
message.updated
message.deleted
typing.started
typing.stopped
conversation.read
```

## Rate Limits

| Group | Limit |
| --- | --- |
| Auth | 20 requests / minute by IP |
| Read API | 3000 requests / minute by user |
| Write API | 600 requests / minute by user |
| Messages | 240 requests / minute by user |

## Testing

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

Docker Compose validation:

```sh
docker compose config --quiet
```

## Notes

- Register returns the created user.
- Login returns the user and JWT.
- Room owners can rename/delete rooms, manage roles, and transfer ownership.
- Admins can add members and remove members.
- Members can leave rooms.
- Owners must transfer ownership before leaving a room.

## Known Limitations

- Email verification is not implemented.
- Password reset is not implemented.
- File/image attachments are not supported.
- Message reactions are not supported.
- Message search is not implemented.
- Presence is based on active WebSocket connections.
- The frontend currently keeps one WebSocket connection per visible conversation.
- Room membership changes are not broadcast as realtime room events yet.
- The app is optimized for local development, not production deployment hardening.

## Future Improvements

- Add email verification and password reset.
- Add file and image message attachments.
- Add message reactions and reply threads.
- Add full-text message search.
- Add realtime room membership events.
- Add push notifications or browser notifications.
- Add profile settings and avatars.
- Add refresh-token based auth.
- Add end-to-end tests for critical chat flows.
- Add production deployment configuration.
