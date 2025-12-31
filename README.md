# Flowday - Go Backend

Flowday is a modern, collaborative platform designed to streamline project management and task tracking. This repository contains the Go Backend, powering the Flowday "Flow State OS" with a high-performance RESTful API.

> [!IMPORTANT]
> **Update (Dec 31, 2024):** Enhanced core collaborative features. Improved the permission system to support shared task management for project members.

## 🛠 Tech Stack

- **Language**: Go (Golang) 1.2+
- **Web Framework**: [Gin](https://github.com/gin-gonic/gin)
- **Database**: [MongoDB](https://www.mongodb.com/) (using official Go driver)
- **Authentication**: JWT ([golang-jwt/jwt/v5](https://github.com/golang-jwt/jwt))
- **Environment Management**: [godotenv](https://github.com/joho/godotenv)
- **Email**: Native SMTP integration for notifications and password resets

## 📁 Project Structure

```text
.
├── server/
│   └── main.go         # Application entry point
├── internal/
    ├── auth/           # Authentication logic (JWT, Password hashing, Handlers)
    ├── db/             # Database connection and collection initialization
    ├── dto/            # Data Transfer Objects
    ├── errors/         # Global error definitions
    ├── handlers/       # Request handlers (Projects, Tasks, Stats, etc.)
    ├── middleware/     # Gin middlewares (Auth, Logging)
    ├── models/         # Database models (BSON/JSON schemas)
    ├── router/         # API route definitions
    └── services/       # Business logic layer (Enhanced Permission Engine)
```

## ⚙️ Setup & Installation

### Prerequisites

- [Go 1.21+](https://go.dev/dl/)
- [MongoDB](https://www.mongodb.com/try/download/community) (Local or Atlas)

### Running the Application

1. Install dependencies:
   ```bash
   go mod download
   ```

2. Run the server:
   ```bash
   go run ./server
   ```

## 📡 API Endpoints & Capabilities

### Enhanced Permission Engine
The backend now features a centralized project access verification system. 
- **Owner**: Full control over projects, members, and tasks.
- **Accepted Member**: Can create, update, and delete tasks within shared projects.

### Authentication
- `POST /api/v1/auth/register` - Create a new account
- `POST /api/v1/auth/login` - Authenticate and get JWT
- `POST /api/v1/auth/forgot-password` - Request a password reset code
- `POST /api/v1/auth/reset-password` - Reset password using code

### Tasks (Protected)
- `GET /api/v1/tasks` - List tasks (filter via `?project_id=`)
- `POST /api/v1/tasks` - Create a new task (Now supports optional deadlines)
- `PATCH /api/v1/tasks/:id` - Update task status/details (Accessible to all project members)
- `DELETE /api/v1/tasks/:id` - Remove a task (Accessible to all project members)
- `GET /api/v1/tasks/by-date` - Get tasks for a specific date
- `GET /api/v1/tasks/by-range` - Get tasks for calendar date ranges
- `GET /api/v1/tasks/stats` - Consolidated dashboard statistics

## 📈 Roadmap (Backend)

- [x] Secure JWT-based Authentication
- [x] Project and Task CRUD logic
- [x] Email Notification System (Invitations & Resets)
- [x] Collaborative Task Management (Member Permissions)
- [x] Calendar Range Aggregations
- [ ] Real-time Collaboration (WebSockets/SSE)
- [ ] File Attachments for tasks
- [ ] Advanced Productivity Analytics
