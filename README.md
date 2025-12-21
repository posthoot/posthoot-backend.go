# 🐦 Xem Backend

> 🚀 A robust Go-based backend service for email campaign management with advanced authentication and permission systems.

## ✨ Features

### 🔐 Authentication System
- 🎯 JWT-based authentication with refresh tokens
- 👥 Role-based access control (RBAC)
- 🔑 Password reset functionality with time-limited codes
- 🔒 Support for API keys with granular permissions
- 👑 Super admin creation on first run

### 🛡️ Permission System
- 📊 Granular resource-based permissions
- 🏗️ Module-based organization
- 👤 Role-based default permissions
- 🌟 Support for wildcard permissions (e.g., "campaigns:*")

### 🎯 Supported Modules

#### 1. 📨 Campaign Management
   - 📝 Create, read, update, delete campaigns
   - ⏰ Campaign scheduling and automation

#### 2. 📋 Template Management
   - 🎨 Email template creation and management
   - 💻 HTML template support

#### 3. 👥 Contact Management
   - 📚 Mailing list management
   - 📥 Contact import/export
   - 🏷️ Contact tagging

#### 4. 🏢 Team Management
   - 🌐 Multi-team support
   - ✉️ Team invitations
   - ⚙️ Team settings

#### 5. 👤 User Management
   - 👑 User roles (Super Admin, Admin, Member)
   - 🔑 Permission management
   - 👤 Profile management

#### 6. 🔑 API Key Management
   - 🎯 Generate and manage API keys
   - 🔒 Granular API permissions
   - 📊 Usage tracking

#### 7. 🤖 Automation
   - ⚡ Email automation workflows
   - 🎯 Trigger-based actions
   - 🧩 Custom automation nodes

#### 8. 📧 SMTP Configuration
   - 🔌 Multiple SMTP provider support
   - ✅ SMTP testing and validation
   - ⚡ Send rate management

#### 9. 🌐 Domain Management
   - ✅ Domain verification
   - 🔧 DNS record management
   - 🌍 Multiple domain support

#### 10. 🔌 Webhook Management
    - 🎯 Custom webhook endpoints
    - ⚡ Event-based triggers
    - 📊 Delivery tracking

### 🚌 Event Bus System
- 🎯 Decoupled service communication
- ⚡ Asynchronous event handling
- 🔌 Service hooks integration
- 🛡️ Panic recovery in event handlers

```mermaid
graph LR
    A[Email Service] -->|Emit| B[Event Bus]
    B -->|Notify| C[Template Service]
    B -->|Notify| D[Campaign Service]
    B -->|Notify| E[Webhook Service]
    
    style A fill:#f9f,stroke:#333,stroke-width:2px
    style B fill:#bbf,stroke:#333,stroke-width:4px
    style C fill:#bfb,stroke:#333,stroke-width:2px
    style D fill:#bfb,stroke:#333,stroke-width:2px
    style E fill:#bfb,stroke:#333,stroke-width:2px
```

#### Event Flow Architecture

```mermaid
sequenceDiagram
    participant M as Models
    participant E as Event Bus
    participant S as Services
    participant H as Hooks
    
    M->>E: Emit Event
    activate E
    E->>S: Notify Service
    E->>H: Trigger Hooks
    S-->>E: Process Event
    H-->>E: Execute Hook
    deactivate E
```

#### Available Events
| Event Name | Description | Payload |
|------------|-------------|---------|
| email.sent | Triggered when email is sent | EmailData |
| template.updated | Triggered on template changes | TemplateData |
| campaign.started | Triggered when campaign starts | CampaignData |
| user.registered | Triggered on new registration | UserData |
| team.created | Triggered when a new team is created | TeamData |

#### Example Usage
```go
// Register event handler
events.On("email.sent", func(data interface{}) {
    // Handle email sent event
})

// Emit event
events.Emit("email.sent", emailData)
```

## 🚀 Getting Started

### 📋 Prerequisites
- 🔧 Go 1.21 or higher
- 🗄️ PostgreSQL 14 or higher
- ⚡ Redis (for rate limiting and caching)

### 🔧 Environment Variables
```env
# 🖥️ Server Configuration
SERVER_HOST=localhost
SERVER_PORT=8080

# 🗄️ Database Configuration
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=kori_user
POSTGRES_PASSWORD=kori_password
POSTGRES_DB=kori
POSTGRES_SSLMODE=disable

# 🔒 JWT Configuration
JWT_SECRET=your_secure_jwt_secret

# 📁 Storage Configuration
STORAGE_PROVIDER=local
STORAGE_BASE_PATH=./storage

# ⚙️ Worker Configuration
WORKER_CONCURRENCY=5
WORKER_QUEUE_SIZE=100

# 🔄 Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=kori_password
REDIS_DB=0

# 👑 Super Admin Configuration (First Run)
SUPERADMIN_EMAIL=admin@example.com
SUPERADMIN_PASSWORD=secure_password
SUPERADMIN_NAME=Admin
```

### 📥 Installation

1. Clone the repository:
```bash
git clone https://github.com/mailexem/Xem.go.git xem
cd xem
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
4. Start the server:
```bash
go run cmd/server/main.go
```

### 📚 API Documentation

The API is documented using Swagger/OpenAPI specification. The documentation provides comprehensive coverage of all endpoints, request/response models, and authentication methods.

#### 🚀 Quick Access

- **Swagger UI**: `http://localhost:8080/swagger/index.html`
- **OpenAPI 3.0 JSON**: `openapi.json` (generated file)
- **Swagger 2.0 JSON**: `http://localhost:8080/swagger/doc.json`
- **Swagger 2.0 YAML**: `http://localhost:8080/swagger/doc.yaml`

#### 🛠️ Documentation Management

Generate or update the API documentation:

```bash
# Generate Swagger 2.0 documentation
make docs

# Generate OpenAPI 3.0 specification (openapi.json)
make openapi

# Using the scripts
./scripts/swagger.sh generate
./scripts/generate-openapi.sh generate

# Serve documentation locally
./scripts/swagger.sh serve
```

#### 📖 Documentation Features

- **Interactive Testing**: Test API endpoints directly from the browser
- **Request/Response Examples**: Pre-filled examples for all endpoints
- **Authentication Support**: Built-in JWT and API key authentication
- **Model Schemas**: Complete request/response model definitions
- **Error Responses**: Detailed error response documentation
- **Tagged Organization**: Endpoints organized by functional areas

#### 🏷️ Available API Categories

- **Authentication** - User registration, login, and token management
- **Teams** - Team management and collaboration features
- **Campaigns** - Email campaign creation, management, and tracking
- **Analytics** - Campaign analytics, audience insights, and performance metrics
- **Contacts** - Contact management and mailing list operations
- **Templates** - Email template management and customization
- **Automations** - Email automation workflows and triggers
- **SMTP** - SMTP configuration and email delivery settings
- **IMAP** - IMAP configuration for email inbox management
- **Webhooks** - Webhook management for real-time event notifications
- **Files** - File upload and management for attachments and media
- **Domains** - Domain management for email authentication
- **API Keys** - API key management for programmatic access

For detailed documentation management, see [`docs/README.md`](docs/README.md).

## 🔐 Authentication

### 📝 Registration
```http
POST /api/v1/auth/register
{
    "email": "user@example.com",
    "password": "secure_password",
    "first_name": "John",
    "last_name": "Doe"
}
```

### 🔑 Login
```http
POST /api/v1/auth/login
{
    "email": "user@example.com",
    "password": "secure_password"
}
```

### 🔄 Password Reset
```http
POST /api/v1/auth/password-reset
{
    "email": "user@example.com"
}
```

### 🔒 Authentication System Architecture

The authentication system supports both traditional email/password authentication and Google OAuth, integrated with JWT-based session management.

```mermaid
graph TD
    subgraph "Traditional Email/Password Authentication"
        A[User Registration/Login] -->|Email & Password| B{Exists?}
        B -->|No - Register| C[Create Team & User]
        C --> D[Assign Default Permissions]
        B -->|Yes - Login| E[Validate Password]
        D --> F[Generate Tokens]
        E -->|Valid| F
        E -->|Invalid| G[Return Error]
        F --> H[Create Auth Transaction]
        H --> I[Return JWT & Refresh Token]
    end

    subgraph "Google OAuth Authentication"
        J[Google Sign-In] -->|ID Token| K[Verify with Firebase]
        K -->|Valid| L{User Exists?}
        L -->|No| M[Create Team & User]
        M --> N[Assign Default Permissions]
        L -->|Yes| O[Update Provider Data]
        N --> P[Generate Tokens]
        O --> P
        P --> Q[Create Auth Transaction]
        Q --> R[Return JWT & Refresh Token]
    end

    subgraph "JWT Token Flow"
        S[Protected API Request] -->|JWT Token| T[Auth Middleware]
        T -->|Validate| U{Token Valid?}
        U -->|Yes| V[Extract Claims]
        V --> W[Set Context]
        W --> X[Continue to Handler]
        U -->|No| Y[Return 401]
    end

    subgraph "Token Refresh Flow"
        Z[Refresh Token Request] -->|Refresh Token| AA{Valid?}
        AA -->|Yes| AB[Get User]
        AB --> AC[Generate New Access Token]
        AC --> AD[Update Auth Transaction]
        AD --> AE[Return New Access Token]
        AA -->|No| AF[Return 401]
    end

    subgraph "Password Reset Flow"
        AG[Reset Request] -->|Email| AH[Generate Reset Code]
        AH --> AI[Store Reset Code]
        AI --> AJ[Send Reset Email]
        AK[Reset Verification] -->|Code & New Password| AL{Valid Code?}
        AL -->|Yes| AM[Update Password]
        AM --> AN[Mark Code Used]
        AL -->|No| AO[Return Error]
    end

    subgraph "Team Invite Flow"
        AP[Team Invite] -->|Email & Role| AQ[Generate Invite Code]
        AQ --> AR[Store Invite]
        AR --> AS[Send Invite Email]
        AT[Accept Invite] -->|Code & Password| AU{Valid Invite?}
        AU -->|Yes| AV[Create User]
        AV --> AW[Assign Team & Role]
        AU -->|No| AX[Return Error]
    end
```

#### Key Components:

1. **🔐 Authentication Methods**
   - 📧 Traditional Email/Password
   - 🔑 Google OAuth via Firebase
   - 📨 Team Invitations

2. **🎟️ Token Management**
   - 🔒 JWT Access Tokens (24h validity)
   - 🔄 Refresh Tokens (7 days validity)
   - 📝 Auth Transaction Tracking

3. **👥 User Management**
   - 🏢 Automatic Team Creation
   - 👑 Role Assignment
   - 🔑 Permission Management

4. **🔒 Security Features**
   - 🔐 Bcrypt Password Hashing
   - ⏰ Time-Limited Reset Codes
   - 🔍 Firebase Token Verification
   - 📊 Transaction-based Operations

5. **🤝 Integration Points**
   - 🔌 Firebase Authentication
   - 📨 Email Service for Notifications
   - 📝 Event System for Tracking

#### Authentication Endpoints:

```http
# Traditional Authentication
POST /api/v1/auth/register     # User Registration
POST /api/v1/auth/login        # User Login
POST /api/v1/auth/refresh      # Token Refresh

# Google OAuth
POST /api/v1/auth/google       # Google Sign-In

# Password Management
POST /api/v1/auth/password-reset         # Request Reset
POST /api/v1/auth/password-reset/verify  # Verify Reset

# Team Management
POST /api/v1/auth/invite       # Send Team Invite
POST /api/v1/auth/accept/:code # Accept Invite
```

### 💳 Subscription System

The subscription system integrates with Dodo Payments to provide flexible subscription management with feature-based access control.

```mermaid
sequenceDiagram
    participant U as User
    participant F as Frontend
    participant B as Backend
    participant D as Dodo Payments
    participant DB as Database

    %% Initial Purchase Flow
    U->>F: Click Buy Now
    F->>B: POST /subscriptions (email + productId)
    B->>D: Create Customer
    B->>D: Create Subscription
    B->>DB: Store Pending Subscription
    B-->>F: Return Payment URL
    F->>D: Redirect to Payment Page
    D->>B: Webhook (subscription.activated)
    B->>DB: Update Subscription Status

    %% User Registration/Login Flow
    U->>F: Register/Login
    F->>B: POST /auth/register or /auth/google
    B->>DB: Check for Pending Subscription
    B->>DB: Link Subscription to Team
    B-->>F: Return JWT Token

    %% Subscription Management Flow
    U->>F: Manage Subscription
    F->>B: GET /subscriptions/portal
    B->>D: Create Portal Session
    B-->>F: Return Portal URL
    F->>D: Redirect to Portal
```

#### 🎯 Key Components

1. **📦 Products & Features**
   ```go
   type Product struct {
       Name        string
       Description string
       Price       float64
       Interval    string    // monthly, yearly
       Features    []ProductFeatureConfig
   }

   type ProductFeature string
   const (
       FeatureEmailCampaigns    ProductFeature = "email_campaigns"
       FeatureTemplateLibrary   ProductFeature = "template_library"
       FeatureAdvancedAnalytics ProductFeature = "advanced_analytics"
       // ... more features
   )
   ```

2. **🔄 Subscription States**
   ```go
   type SubscriptionStatus string
   const (
       SubscriptionStatusPending  SubscriptionStatus = "pending"
       SubscriptionStatusActive   SubscriptionStatus = "active"
       SubscriptionStatusCanceled SubscriptionStatus = "canceled"
       SubscriptionStatusPaused   SubscriptionStatus = "paused"
       SubscriptionStatusFailed   SubscriptionStatus = "failed"
   )
   ```

#### 🛣️ Subscription Flow

1. **💰 Pre-Purchase**
   - User selects a plan
   - Backend creates pending subscription
   - User is redirected to Dodo Payments

2. **👤 Account Creation**
   - User registers/logs in after payment
   - System matches email with pending subscription
   - Subscription is linked to user's team

3. **✨ Feature Access**
   - Each product defines enabled features
   - System checks feature access via `HasFeature()`
   - Optional limits per feature (e.g., email campaign limits)

4. **⚙️ Management**
   - Team admins can access subscription portal
   - Portal allows plan changes, cancellation
   - Webhooks handle subscription updates

#### 🔌 API Endpoints

```http
# Public Endpoints
POST   /api/v1/subscriptions         # Create subscription
POST   /api/v1/subscriptions/webhook # Handle Dodo webhooks

# Protected Endpoints (Requires Auth)
GET    /api/v1/subscriptions/portal   # Get management portal URL
GET    /api/v1/subscriptions/features # Get enabled features
```

#### 🔐 Security Features

1. **👥 Access Control**
   - Only team admins can manage subscriptions
   - Feature checks on all protected endpoints
   - Webhook signature verification

2. **💾 Data Integrity**
   - Transaction-based subscription updates
   - Email verification for subscription linking
   - Secure portal access via Dodo Payments

3. **🔄 State Management**
   - Automatic status updates via webhooks
   - Period tracking for billing cycles
   - Trial period support

#### ⚙️ Configuration

```env
# Dodo Payments Configuration
DODO_API_KEY=your_dodo_api_key
DODO_WEBHOOK_SECRET=your_dodo_webhook_secret
APP_ENV=development # or production
```

## 🛡️ Security Features

1. **⚡ Rate Limiting**
   - 🔒 Request rate limiting per IP
   - 🔑 API key rate limiting
   - ⚙️ Configurable limits

2. **🔒 JWT Security**
   - ⏱️ Short-lived access tokens (24 hours)
   - 🔄 Refresh token support (7 days)
   - 🎯 Permission claims in tokens

3. **🔐 Password Security**
   - 🔒 Bcrypt password hashing
   - ✅ Minimum password requirements
   - 🛡️ Secure password reset flow

4. **🔒 API Security**
   - 🌐 CORS protection
   - 📦 Request size limiting
   - 🛡️ Secure headers
   - 🗜️ GZIP compression

## 👨‍💻 Development

### 📁 Project Structure
```
📦 xem
 ┣ 📂 cmd                     # Application entry points
 ┣ 📂 internal               
 ┃ ┣ 📂 api                  # API layer
 ┃ ┃ ┣ 📂 middleware         # Custom middlewares
 ┃ ┃ ┣ 📂 validator          # Request validators
 ┃ ┃ ┗ 📜 server.go          # Server setup
 ┃ ┣ 📂 config               # Configuration
 ┃ ┣ 📂 events               # Event bus system
 ┃ ┣ 📂 handlers             # Request handlers
 ┃ ┣ 📂 models               # Database models
 ┃ ┣ 📂 routes               # Route definitions
 ┃ ┣ 📂 services             # Business logic
 ┃ ┗ 📂 utils                # Utility functions
 ┣ 📂 migrations             # Database migrations
 ┗ 📂 storage                # Local storage
```

### ✨ Adding New Features

1. **📦 New Resource**
   - 📝 Add model in `internal/models/`
   - 🔑 Add permissions in `internal/models/seed.go`
   - 🎯 Create handler in `internal/handlers/`
   - 🔌 Add routes in `internal/routes/`

2. **🔑 New Permission**
   - 📝 Add resource in `defaultResources`
   - 👥 Add permissions in `rolePermissions`
   - 🔄 Run server to auto-seed

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details. 
