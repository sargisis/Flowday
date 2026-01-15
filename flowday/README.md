# Flowday Backend API

**Production-ready Go backend** for Flowday - a productivity and task management platform with AI-powered features.

## 🚀 Features

### Performance
- ✅ **Gzip compression** - Automatic response compression (60-80% size reduction)
- ✅ **Database indexes** - 50+ optimized indexes for fast queries
- ✅ **Connection pooling** - MongoDB connection pool (100 max, 10 min)
- ✅ **Caching** - In-memory cache with Redis support ready
- ✅ **Request timeouts** - Configurable timeouts to prevent hanging requests

### Stability
- ✅ **Graceful shutdown** - Clean shutdown with 30s timeout
- ✅ **Health checks** - `/health`, `/ready`, `/live` endpoints
- ✅ **Panic recovery** - Automatic panic recovery with logging
- ✅ **Server timeouts** - Read/Write/Idle timeouts configured

### Security
- ✅ **JWT authentication** - Secure token-based auth with refresh tokens
- ✅ **Rate limiting** - Global (100/min per IP) + User-based (200/min per user)
- ✅ **Security headers** - X-Frame-Options, CSP, HSTS, etc.
- ✅ **Input sanitization** - Protection against injection attacks
- ✅ **File validation** - MIME type, size, and extension validation
- ✅ **Protected static files** - Authorization required for file access
- ✅ **Audit logging** - Logging of all important user actions

### Monitoring & Observability
- ✅ **Structured logging** - JSON logs in production, readable in development
- ✅ **Prometheus metrics** - Comprehensive metrics collection
- ✅ **Request ID tracking** - Full request tracing
- ✅ **Error handling** - Centralized error handling with proper responses

### Additional Features
- ✅ **Email templates** - Beautiful, responsive HTML email templates
- ✅ **Circuit breaker** - Protection against cascading failures (ready for external services)
- ✅ **API documentation** - Swagger/OpenAPI documentation (ready)

## 📋 Requirements

- Go 1.21+
- MongoDB 4.4+
- (Optional) Redis for distributed rate limiting

## 🛠️ Installation

```bash
# Clone the repository
git clone <repository-url>
cd flowday/flowday

# Install dependencies
go mod download

# Copy environment file
cp .env.example .env

# Edit .env with your configuration
nano .env
```

## ⚙️ Configuration

### Required Environment Variables

```env
# Database
MONGO_URI=mongodb://localhost:27017

# JWT Secrets (minimum 32 characters each)
JWT_SECRET=your-super-secret-jwt-key-minimum-32-chars
JWT_REFRESH_SECRET=your-super-secret-refresh-key-minimum-32-chars

# Email (SMTP)
SMTP_EMAIL=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587

# Application
APP_NAME=Flowday
GIN_MODE=release  # or debug for development
LOG_LEVEL=info     # debug, info, warn, error
```

### Optional Environment Variables

```env
# CORS
ALLOWED_ORIGINS=https://app.flowday.com,https://www.flowday.com

# Rate Limiting (Redis)
REDIS_URL=redis://localhost:6379

# Request Timeout
REQUEST_TIMEOUT=30s

# Slack Integration
SLACK_CLIENT_ID=your-slack-client-id
SLACK_CLIENT_SECRET=your-slack-client-secret
SLACK_REDIRECT_URI=https://your-domain.com/api/v1/slack/oauth/callback

# AI Service (Groq)
GROQ_API_KEY=your-groq-api-key
```

## 🏃 Running the Server

```bash
# Development mode
go run server/main.go

# Production mode
GIN_MODE=release go run server/main.go

# Or build and run
go build -o flowday server/main.go
./flowday
```

The server will start on `http://localhost:8080`

## 📚 API Endpoints

### Health & Monitoring

- `GET /health` - Health check endpoint
- `GET /ready` - Readiness check (for Kubernetes)
- `GET /live` - Liveness check (for Kubernetes)
- `GET /metrics` - Prometheus metrics

### Authentication

- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - Login and get JWT tokens
- `POST /api/v1/auth/refresh` - Refresh access token
- `POST /api/v1/auth/logout` - Logout (revoke refresh token)
- `POST /api/v1/auth/forgot-password` - Request password reset
- `POST /api/v1/auth/reset-password` - Reset password with code

### Users

- `GET /api/v1/me` - Get current user profile
- `PATCH /api/v1/users/profile` - Update user profile
- `POST /api/v1/users/avatar` - Upload avatar
- `POST /api/v1/users/email-change/request` - Request email change
- `POST /api/v1/users/change-email/verify` - Verify email change

### Projects

- `GET /api/v1/projects` - List user's projects
- `POST /api/v1/projects` - Create project
- `DELETE /api/v1/projects/:id` - Delete project

### Tasks

- `GET /api/v1/tasks` - List tasks (with filters)
- `POST /api/v1/tasks` - Create task
- `PATCH /api/v1/tasks/:id` - Update task
- `DELETE /api/v1/tasks/:id` - Delete task
- `GET /api/v1/tasks/search` - Search tasks
- `GET /api/v1/tasks/by-date` - Get tasks by date
- `GET /api/v1/tasks/by-range` - Get tasks by date range

### AI Features

- `POST /api/v1/ai/chat` - AI chat
- `POST /api/v1/ai/health-advice` - Get health advice
- `GET /api/v1/ai/quota` - Get AI quota
- `GET /api/v1/ai/insights` - Get AI insights

See `/docs/swagger.yaml` for complete API documentation.

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test ./... -cover

# Run specific package tests
go test ./internal/middleware/... -v

# Run tests with race detection
go test ./... -race
```

### Test Coverage

- ✅ Middleware tests (logging, rate limiting, request ID)
- ✅ Circuit breaker tests
- ✅ More tests coming soon...

## 📊 Metrics

Prometheus metrics are available at `/metrics`:

- `http_requests_total` - Total HTTP requests
- `http_request_duration_seconds` - Request duration
- `http_request_size_bytes` - Request size
- `http_response_size_bytes` - Response size
- `database_operations_total` - Database operations
- `auth_attempts_total` - Authentication attempts
- `rate_limit_hits_total` - Rate limit hits

## 🔒 Security Features

### Rate Limiting
- **Global**: 100 requests/minute per IP
- **Per User**: 200 requests/minute per authenticated user
- **Auth endpoints**: 5 requests/minute (stricter)

### Security Headers
- `X-Frame-Options: DENY`
- `X-Content-Type-Options: nosniff`
- `X-XSS-Protection: 1; mode=block`
- `Content-Security-Policy`
- `Strict-Transport-Security` (production only)

### File Upload Security
- MIME type validation
- File size limits (5MB avatars, 10MB attachments)
- Extension validation
- Safe filename generation

## 📝 Logging

### Log Levels
- `DEBUG` - Detailed information for debugging
- `INFO` - General informational messages
- `WARN` - Warning messages
- `ERROR` - Error messages

### Log Format
- **Development**: Human-readable text format
- **Production**: JSON format for log aggregation

### Audit Logging
All important actions are logged with:
- User ID
- Action type
- IP address
- Request ID
- Timestamp

## 🏗️ Architecture

```
flowday/
├── server/
│   └── main.go              # Application entry point
├── internal/
│   ├── auth/                # Authentication & authorization
│   ├── cache/               # Caching layer
│   ├── circuitbreaker/      # Circuit breaker for external services
│   ├── db/                  # Database connection & indexes
│   ├── handlers/            # HTTP request handlers
│   ├── logger/              # Structured logging
│   ├── metrics/             # Prometheus metrics
│   ├── middleware/          # HTTP middleware
│   ├── models/              # Data models
│   ├── router/              # Route definitions
│   ├── services/            # Business logic
│   └── utils/               # Utility functions
└── docs/                    # API documentation
```

## 🚢 Deployment

### Docker (Recommended)

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o flowday server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/flowday .
CMD ["./flowday"]
```

### Environment Setup

1. Set all required environment variables
2. Ensure MongoDB is accessible
3. (Optional) Set up Redis for distributed rate limiting
4. Configure reverse proxy (nginx/traefik) with SSL
5. Set `GIN_MODE=release` for production

## 📈 Performance Tips

1. **Use Redis** for rate limiting in production clusters
2. **Enable caching** for frequently accessed data
3. **Monitor metrics** via Prometheus
4. **Set appropriate timeouts** based on your use case
5. **Use connection pooling** (already configured)

## 🐛 Troubleshooting

### Server won't start
- Check that `JWT_SECRET` and `JWT_REFRESH_SECRET` are set (min 32 chars)
- Verify MongoDB connection
- Check logs for specific errors

### Rate limiting issues
- Check `X-RateLimit-*` headers in responses
- Verify Redis connection if using distributed rate limiting

### Database slow queries
- Check that indexes are created (see logs on startup)
- Use MongoDB explain() to analyze queries

## 📄 License

MIT License - see LICENSE file for details

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Write tests for your changes
4. Ensure all tests pass
5. Submit a pull request

## 📞 Support

For issues and questions:
- Email: support@flowday.com
- Documentation: `/docs/swagger.yaml`

---

**Built with ❤️ using Go, Gin, and MongoDB**
