# Flowday - Go Backend

Flowday is a modern, collaborative platform designed to streamline project management and task tracking. This directory contains the Go Backend, which powers the Flowday application with a robust API.

> [!NOTE]
> This project is currently in pre-v1 status. Many core features are implemented, but refinement and final stabilizing work are ongoing.

## 🛠 Tech Stack

- **Language**: Go (Golang) 1.25+
- **Web Framework**: [Gin](https://github.com/gin-gonic/gin)
- **Database**: [MongoDB](https://www.mongodb.com/) (using official Go driver)
- **Authentication**: JWT ([golang-jwt/jwt/v5](https://github.com/golang-jwt/jwt))
- **Environment Management**: [godotenv](https://github.com/joho/godotenv)
- **Email**: Native SMTP integration for notifications and password resets

## 📁 Project Structure

The project code is located in the `flowday/` subdirectory:

```text
.
├── flowday/
│   ├── .env                # Environment variables (create this)
│   ├── go.mod              # Go module definition
│   ├── server/
│   │   └── main.go         # Application entry point
│   └── internal/
│       ├── auth/           # Authentication logic (JWT, Password hashing)
│       ├── db/             # Database connection and collections
│       ├── handlers/       # HTTP Request handlers
│       ├── middleware/     # Auth and logging middlewares
│       ├── models/         # Database models (BSON/JSON schemas)
│       ├── router/         # API route definitions
│       └── services/       # Business logic (Emails, Invites, etc.)
```

## ⚙️ Environment Configuration

Create a `.env` file in the `flowday/` directory with the following variables:

| Variable | Description | Default / Example |
|----------|-------------|-------------------|
| `MONGO_URI` | MongoDB connection string | `mongodb://localhost:27017` |
| `JWT_SECRET` | Secret key for signing tokens | `super-secret-key` |
| `SMTP_HOST` | SMTP server host | `smtp.example.com` |
| `SMTP_PORT` | SMTP server port | `587` |
| `SMTP_EMAIL` | Email address for sending notifications | `no-reply@flowday.app` |
| `SMTP_PASSWORD` | Password for the SMTP email | `your-smtp-password` |

## ⚙️ Setup & Installation

### Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [MongoDB](https://www.mongodb.com/try/download/community) (Local or Atlas)

### Running the Application

1. Navigate to the project directory:
   ```bash
   cd flowday
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Run the server:
   ```bash
   go run ./server
   ```

   The server will start on port **8080**.
   It is configured to accept CORS requests from `http://localhost:5173` (Vite Frontend).

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
