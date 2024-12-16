# Kori Backend

A robust Go-based backend service for email campaign management with advanced authentication and permission systems.

## Features

### Authentication System
- JWT-based authentication with refresh tokens
- Role-based access control (RBAC)
- Password reset functionality with time-limited codes
- Support for API keys with granular permissions
- Super admin creation on first run

### Permission System
- Granular resource-based permissions
- Module-based organization
- Role-based default permissions
- Support for wildcard permissions (e.g., "campaigns:*")

### Supported Modules
1. **Campaign Management**
   - Create, read, update, delete campaigns
   - Campaign scheduling and automation

2. **Template Management**
   - Email template creation and management
   - HTML template support

3. **Contact Management**
   - Mailing list management
   - Contact import/export
   - Contact tagging

4. **Team Management**
   - Multi-team support
   - Team invitations
   - Team settings

5. **User Management**
   - User roles (Super Admin, Admin, Member)
   - Permission management
   - Profile management

6. **API Key Management**
   - Generate and manage API keys
   - Granular API permissions
   - Usage tracking

7. **Automation**
   - Email automation workflows
   - Trigger-based actions
   - Custom automation nodes

8. **SMTP Configuration**
   - Multiple SMTP provider support
   - SMTP testing and validation
   - Send rate management

9. **Domain Management**
   - Domain verification
   - DNS record management
   - Multiple domain support

10. **Webhook Management**
    - Custom webhook endpoints
    - Event-based triggers
    - Delivery tracking

## Getting Started

### Prerequisites
- Go 1.21 or higher
- PostgreSQL 14 or higher
- Redis (for rate limiting and caching)

### Environment Variables
```env
# Server Configuration
SERVER_HOST=localhost
SERVER_PORT=8080

# Database Configuration
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=kori_user
POSTGRES_PASSWORD=kori_password
POSTGRES_DB=kori
POSTGRES_SSLMODE=disable

# JWT Configuration
JWT_SECRET=your_secure_jwt_secret

# Storage Configuration
STORAGE_PROVIDER=local
STORAGE_BASE_PATH=./storage

# Worker Configuration
WORKER_CONCURRENCY=5
WORKER_QUEUE_SIZE=100

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=kori_password
REDIS_DB=0

# Super Admin Configuration (First Run)
SUPERADMIN_EMAIL=admin@example.com
SUPERADMIN_PASSWORD=secure_password
SUPERADMIN_NAME=Admin
```

### Installation

1. Clone the repository:
```bash
git clone https://github.com/yourusername/kori.git
cd kori
```

2. Install dependencies:
```bash
go mod download
```

3. Set up the environment:
```bash
cp .env.example .env
# Edit .env with your configuration
```

4. Run migrations:
```bash
go run cmd/migrate/main.go
```

5. Start the server:
```bash
go run cmd/server/main.go
```

### API Documentation

The API is documented using Swagger/OpenAPI. Access the documentation at:
```
http://localhost:8080/swagger/index.html
```

## Authentication

### Registration
```http
POST /api/v1/auth/register
{
    "email": "user@example.com",
    "password": "secure_password",
    "first_name": "John",
    "last_name": "Doe"
}
```

### Login
```http
POST /api/v1/auth/login
{
    "email": "user@example.com",
    "password": "secure_password"
}
```

### Password Reset
```http
POST /api/v1/auth/password-reset
{
    "email": "user@example.com"
}
```

## Security Features

1. **Rate Limiting**
   - Request rate limiting per IP
   - API key rate limiting
   - Configurable limits

2. **JWT Security**
   - Short-lived access tokens (24 hours)
   - Refresh token support (7 days)
   - Permission claims in tokens

3. **Password Security**
   - Bcrypt password hashing
   - Minimum password requirements
   - Secure password reset flow

4. **API Security**
   - CORS protection
   - Request size limiting
   - Secure headers
   - GZIP compression

## Development

### Project Structure
```
├── cmd/                    # Application entry points
├── internal/              
│   ├── api/               # API layer
│   │   ├── middleware/    # Custom middlewares
│   │   ├── validator/     # Request validators
│   │   └── server.go      # Server setup
│   ├── config/            # Configuration
│   ├── handlers/          # Request handlers
│   ├── models/            # Database models
│   ├── routes/            # Route definitions
│   ├── services/          # Business logic
│   └── utils/             # Utility functions
├── migrations/            # Database migrations
└── storage/              # Local storage
```

### Adding New Features

1. **New Resource**
   - Add model in `internal/models/`
   - Add permissions in `internal/models/seed.go`
   - Create handler in `internal/handlers/`
   - Add routes in `internal/routes/`

2. **New Permission**
   - Add resource in `defaultResources`
   - Add permissions in `rolePermissions`
   - Run server to auto-seed

## License

This project is licensed under the MIT License - see the LICENSE file for details. 