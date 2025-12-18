# Flowday - Go Backend

Flowday is a modern, collaborative platform designed to streamline project management and task tracking. This directory contains the Go Backend, which powers the Flowday application with a robust API.

> [!NOTE]
> This project is currently in pre-v1 status. Many core features are implemented, but refinement and final stabilizing work are ongoing.

## 🛠 Tech Stack

- **Language**: Go (Golang) 1.25+
- **Web Framework**: [Gin](https://github.com/gin-gonic/gin)
- **Database**: [MongoDB](https://www.mongodb.com/) (using official Go driver)
- **Database (Migration/Other)**: [GORM](https://gorm.io/) with SQLite support
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
   └── services/       # Business logic layer
```

## ⚙️ Setup & Installation

### Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [MongoDB](https://www.mongodb.com/try/download/community) (Local or Atlas)

### Environment Configuration

### Running the Application

1. Install dependencies:
   ```bash
   go mod download
   ```

2. Run the server:
   ```bash
   go run ./server
   ```

## 📡 API Endpoints

### Authentication
- `POST /api/v1/auth/register` - Create a new account
- `POST /api/v1/auth/login` - Authenticate and get JWT
- `POST /api/v1/auth/forgot-password` - Request a password reset code
- `POST /api/v1/auth/reset-password` - Reset password using code

### User
- `GET /api/v1/me` - Get current user info (Protected)

### Projects (Protected)
- `GET /api/v1/projects` - List all projects
- `POST /api/v1/projects` - Create a new project
- `DELETE /api/v1/projects/:id` - Delete a project

### Tasks (Protected)
- `GET /api/v1/tasks` - List tasks (filter via `?project_id=`)
- `POST /api/v1/tasks` - Create a new task
- `PATCH /api/v1/tasks/:id` - Update task status/details
- `DELETE /api/v1/tasks/:id` - Remove a task
- `GET /api/v1/tasks/by-date` - Get tasks for a specific date (`?date=YYYY-MM-DD`)
- `GET /api/v1/tasks/by-range` - Get tasks for a range (`?from=...&to=...`)
- `GET /api/v1/tasks/stats` - Get summary statistics for tasks

### Invitations & Members (Protected)
- `GET /api/v1/invitations` - View pending invitations
- `POST /api/v1/invitations/:id/accept` - Accept a project invitation
- `POST /api/v1/invitations/:id/reject` - Reject a project invitation
- `GET /api/v1/projects/:id/members` - List project members
- `POST /api/v1/projects/:id/members` - Invite a user to a project
- `DELETE /api/v1/projects/:id/members/:userID` - Remove a member from a project

## 📈 Roadmap (Backend)

- [x] Secure JWT-based Authentication
- [x] Project and Task CRUD logic
- [x] Email Notification System (Invitations & Resets)
- [x] Statistics & Calendar Aggregations
- [ ] Drag-and-drop support (REST API optimization)
- [ ] Real-time Collaboration (WebSockets)
- [ ] Advanced Dashboard Analytics
