package docs

// @title Posthoot API
// @version 1.0
// @description Comprehensive API server for Posthoot email marketing platform. Provides endpoints for email campaigns, analytics, user management, team collaboration, and automation workflows.
// @termsOfService https://posthoot.com/terms

// @contact.name Posthoot API Support
// @contact.url https://posthoot.com/support
// @contact.email api-support@posthoot.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host backyard.posthoot.com
// @BasePath /api/v1
// @schemes https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter the token with the `Bearer: ` prefix, e.g. "Bearer abcde12345".

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-KEY
// @description API key for programmatic access

// @tag.name Authentication
// @tag.description User authentication and authorization endpoints

// @tag.name Teams
// @tag.description Team management and collaboration features

// @tag.name Campaigns
// @tag.description Email campaign creation, management, and tracking

// @tag.name Analytics
// @tag.description Campaign analytics, audience insights, and performance metrics

// @tag.name Contacts
// @tag.description Contact management and mailing list operations

// @tag.name Templates
// @tag.description Email template management and customization

// @tag.name Automations
// @tag.description Email automation workflows and triggers

// @tag.name SMTP
// @tag.description SMTP configuration and email delivery settings

// @tag.name IMAP
// @tag.description IMAP configuration for email inbox management

// @tag.name Webhooks
// @tag.description Webhook management for real-time event notifications

// @tag.name Files
// @tag.description File upload and management for attachments and media

// @tag.name Domains
// @tag.description Domain management for email authentication

// @tag.name API Keys
// @tag.description API key management for programmatic access
