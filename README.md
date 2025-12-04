# Telegram Mini App Backend Starter

A robust and scalable backend starter template for building Telegram Mini Apps with Go (Golang). This project provides a solid foundation with built-in Telegram authentication, secure session management using Redis, and JWT-based authorization.

## 🚀 Features

- **Telegram Authentication**: Securely validates Telegram Web App `initData` to authenticate users.
- **Session Management**: Uses Redis to store and manage user sessions with automatic expiration.
- **JWT Authorization**: Issues JSON Web Tokens (JWT) for secure API access after initial authentication.
- **Clean Architecture**: Follows standard Go project layout for maintainability and scalability.
- **Docker Ready**: Includes Dockerfile and docker-compose for easy deployment.

## 🛠 Tech Stack

- **Language**: [Go](https://golang.org/) (1.24+)
- **Framework**: [Gin Web Framework](https://github.com/gin-gonic/gin)
- **Database**: [Redis](https://redis.io/) (for session storage)
- **Authentication**: Telegram `initData` validation & JWT
- **Utilities**: `google/uuid`, `joho/godotenv`

## 📋 Prerequisites

Before you begin, ensure you have the following installed:

- [Go](https://go.dev/dl/) (version 1.24 or higher)
- [Redis](https://redis.io/docs/getting-started/) (running locally or accessible via URL)
- A Telegram Bot Token (get it from [@BotFather](https://t.me/BotFather))

## ⚙️ Configuration

The application is configured via environment variables. You can create a `.env` file in the root directory (see `.env.example` if available, or use the reference below).

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | The port the server listens on | `8080` |
| `ENV` | Environment mode (e.g., `development`, `production`) | - |
| `TELEGRAM_BOT_TOKEN` | **Required**. Your Telegram Bot Token | - |
| `JWT_SECRET` | **Required**. Secret key for signing JWTs | - |
| `JWT_EXPIRY` | JWT expiration duration (e.g., `24h`) | `24h` |
| `REDIS_URL` | Redis connection address | `localhost:6379` |
| `REDIS_PASSWORD` | Redis password (if any) | - |
| `REDIS_DB` | Redis Database index | `0` |

## 🚀 Getting Started

1.  **Clone the repository**
    ```bash
    git clone git@github.com:mikiasyonas/Sample-Telegram-Mini-App-Backend.git
    cd sample-mini-app-backend
    ```

2.  **Install dependencies**
    ```bash
    go mod download
    ```

3.  **Set up configuration**
    Create a `.env` file and populate it with your variables:
    ```env
    PORT=8080
    TELEGRAM_BOT_TOKEN=your_bot_token_here
    JWT_SECRET=your_super_secret_key
    REDIS_URL=localhost:6379
    ```

4.  **Run the application**
    ```bash
    go run cmd/main.go
    ```
    *Note: Adjust the path to `main.go` if it's located elsewhere, e.g., `cmd/api/main.go` or root.*

5.  **Run with Docker** (Optional)
    ```bash
    docker-compose up --build
    ```

## 🔌 API Endpoints

### Authentication

**POST** `/auth/telegram`

Exchange Telegram `initData` for a session JWT.

-   **Query Parameters**:
    -   `initData`: The raw `initData` string from Telegram Web App.
-   **Response**:
    ```json
    {
      "token": "eyJhbGciOiJIUzI1Ni...",
      "expires": 1733400000,
      "user": {
        "id": 123456789,
        "first_name": "John",
        "last_name": "Doe",
        "username": "johndoe",
        "language_code": "en",
        "is_premium": true,
        "photo_url": "..."
      }
    }
    ```

## 📂 Project Structure

```
.
├── cmd/                # Application entry points
├── internal/
│   ├── config/         # Configuration loading
│   ├── handlers/       # HTTP request handlers
│   ├── middleware/     # Gin middleware (Auth, etc.)
│   ├── models/         # Data structures
│   ├── services/       # Business logic (JWT, Redis)
│   └── utils/          # Utility functions (Telegram validation)
├── go.mod              # Go module definition
└── ...
```
