package registry

import (
	"github.com/labstack/echo/v4"

	"kori/internal/api/controllers"
	"kori/internal/api/middleware"
	"kori/internal/models"
	"kori/internal/services"

	"gorm.io/gorm"
)

// RegisterCRUDRoutes registers CRUD routes for all models
func RegisterCRUDRoutes(g *echo.Group, db *gorm.DB) {
	// Teams
	teamService := services.NewBaseService(db, models.Team{})
	teamController := controllers.NewBaseController(teamService)
	teamGroup := g.Group("/teams")
	teamGroup.Use(middleware.RequirePermissions(db, "teams:read"))
	teamGroup.GET("", teamController.List)
	teamGroup.GET("/:id", teamController.Get)

	// Protected team routes
	teamWriteGroup := teamGroup.Group("")
	teamWriteGroup.Use(middleware.RequirePermissions(db, "teams:write"))
	teamWriteGroup.POST("", teamController.Create)
	teamWriteGroup.PUT("/:id", teamController.Update)
	teamWriteGroup.DELETE("/:id", teamController.Delete)

	// Team Invitations with team-specific permissions
	invitationService := services.NewBaseService(db, models.TeamInvite{})
	invitationController := controllers.NewBaseController(invitationService)
	invitationGroup := g.Group("/team-invitations")
	invitationGroup.Use(middleware.RequirePermissions(db, "team_invites:read"))
	invitationGroup.GET("", invitationController.List)

	// Protected invitation routes
	invitationWriteGroup := invitationGroup.Group("")
	invitationWriteGroup.Use(middleware.RequirePermissions(db, "team_invites:write"))
	invitationWriteGroup.DELETE("/:id", invitationController.Delete)

	// file routes
	fileService := services.NewBaseService(db, models.File{})
	fileController := controllers.NewBaseController(fileService)
	fileGroup := g.Group("/files")
	fileGroup.Use(middleware.RequirePermissions(db, "files:read"))
	fileGroup.GET("", fileController.List)
	fileGroup.GET("/:id", fileController.Get)

	// Contacts with team-specific permissions
	contactService := services.NewBaseService(db, models.Contact{})
	contactController := controllers.NewBaseController(contactService)
	contactGroup := g.Group("/contacts")
	contactGroup.Use(middleware.RequirePermissions(db, "contacts:read"))
	contactGroup.GET("", contactController.List)
	contactGroup.GET("/:id", contactController.Get)

	// Protected contact routes
	contactWriteGroup := contactGroup.Group("")
	contactWriteGroup.Use(middleware.RequirePermissions(db, "contacts:write"))
	contactWriteGroup.POST("", contactController.Create)
	contactWriteGroup.PUT("/:id", contactController.Update)
	contactWriteGroup.DELETE("/:id", contactController.Delete)

	// Email Categories with team-specific permissions
	categoryService := services.NewBaseService(db, models.EmailCategory{})
	categoryController := controllers.NewBaseController(categoryService)
	categoryGroup := g.Group("/categories")
	categoryGroup.Use(middleware.RequirePermissions(db, "categories:read"))
	categoryGroup.GET("", categoryController.List)
	categoryGroup.GET("/:id", categoryController.Get)

	// Protected category routes
	categoryWriteGroup := categoryGroup.Group("")
	categoryWriteGroup.Use(middleware.RequirePermissions(db, "categories:write"))
	categoryWriteGroup.POST("", categoryController.Create)
	categoryWriteGroup.PUT("/:id", categoryController.Update)
	categoryWriteGroup.DELETE("/:id", categoryController.Delete)

	// Mailing Lists with team-specific permissions
	mailingListService := services.NewBaseService(db, models.MailingList{})
	mailingListController := controllers.NewBaseController(mailingListService)
	listGroup := g.Group("/mailing-lists")
	listGroup.Use(middleware.RequirePermissions(db, "lists:read"))
	listGroup.GET("", mailingListController.List)
	listGroup.GET("/:id", mailingListController.Get)

	// Protected mailing list routes
	listWriteGroup := listGroup.Group("")
	listWriteGroup.Use(middleware.RequirePermissions(db, "lists:write"))
	listWriteGroup.POST("", mailingListController.Create)
	listWriteGroup.PUT("/:id", mailingListController.Update)
	listWriteGroup.DELETE("/:id", mailingListController.Delete)

	// SMTP Configs with team-specific permissions
	smtpConfigService := services.NewBaseService(db, models.SMTPConfig{})
	smtpConfigController := controllers.NewBaseController(smtpConfigService)
	smtpGroup := g.Group("/smtp-configs")
	smtpGroup.Use(middleware.RequirePermissions(db, "smtp_configs:read"))
	smtpGroup.GET("", smtpConfigController.List)
	smtpGroup.GET("/:id", smtpConfigController.Get)

	// Protected SMTP config routes
	smtpWriteGroup := smtpGroup.Group("")
	smtpWriteGroup.Use(middleware.RequirePermissions(db, "smtp_configs:write"))
	smtpWriteGroup.POST("", smtpConfigController.Create)
	smtpWriteGroup.PUT("/:id", smtpConfigController.Update)
	smtpWriteGroup.DELETE("/:id", smtpConfigController.Delete)

	// Domains with team-specific permissions
	domainService := services.NewBaseService(db, models.Domain{})
	domainController := controllers.NewBaseController(domainService)
	domainGroup := g.Group("/domains")
	domainGroup.Use(middleware.RequirePermissions(db, "domains:read"))
	domainGroup.GET("", domainController.List)
	domainGroup.GET("/:id", domainController.Get)

	// Protected domain routes
	domainWriteGroup := domainGroup.Group("")
	domainWriteGroup.Use(middleware.RequirePermissions(db, "domains:write"))
	domainWriteGroup.POST("", domainController.Create)
	domainWriteGroup.PUT("/:id", domainController.Update)
	domainWriteGroup.DELETE("/:id", domainController.Delete)

	// Webhooks with team-specific permissions
	webhookService := services.NewBaseService(db, models.Webhook{})
	webhookController := controllers.NewBaseController(webhookService)
	webhookGroup := g.Group("/webhooks")
	webhookGroup.Use(middleware.RequirePermissions(db, "webhooks:read"))
	webhookGroup.GET("", webhookController.List)
	webhookGroup.GET("/:id", webhookController.Get)

	// Protected webhook routes
	webhookWriteGroup := webhookGroup.Group("")
	webhookWriteGroup.Use(middleware.RequirePermissions(db, "webhooks:write"))
	webhookWriteGroup.POST("", webhookController.Create)
	webhookWriteGroup.PUT("/:id", webhookController.Update)
	webhookWriteGroup.DELETE("/:id", webhookController.Delete)

	// Templates with team-specific permissions
	templateService := services.NewBaseService(db, models.Template{})
	templateController := controllers.NewBaseController(templateService)
	templateGroup := g.Group("/templates")
	templateGroup.Use(middleware.RequirePermissions(db, "templates:read"))
	templateGroup.GET("", templateController.List)
	templateGroup.GET("/:id", templateController.Get)

	// Protected template routes
	templateWriteGroup := templateGroup.Group("")
	templateWriteGroup.Use(middleware.RequirePermissions(db, "templates:write"))
	templateWriteGroup.POST("", templateController.Create)
	templateWriteGroup.PUT("/:id", templateController.Update)
	templateWriteGroup.DELETE("/:id", templateController.Delete)

	// API Keys with team-specific permissions
	apiKeyService := services.NewBaseService(db, models.APIKey{})
	apiKeyController := controllers.NewBaseController(apiKeyService)
	apiKeyGroup := g.Group("/api-keys")
	apiKeyGroup.Use(middleware.RequirePermissions(db, "api_keys:read"))
	apiKeyGroup.GET("", apiKeyController.List)
	apiKeyGroup.GET("/:id", apiKeyController.Get)

	// Protected API key routes
	apiKeyWriteGroup := apiKeyGroup.Group("")
	apiKeyWriteGroup.Use(middleware.RequirePermissions(db, "api_keys:write"))
	apiKeyWriteGroup.POST("", apiKeyController.Create)
	apiKeyWriteGroup.PUT("/:id", apiKeyController.Update)
	apiKeyWriteGroup.DELETE("/:id", apiKeyController.Delete)

	// API KEY USAGE with team-specific permissions
	apiKeyUsageService := services.NewBaseService(db, models.APIKeyUsage{})
	apiKeyUsageController := controllers.NewBaseController(apiKeyUsageService)
	apiKeyUsageGroup := g.Group("/api-key-usage")
	apiKeyUsageGroup.Use(middleware.RequirePermissions(db, "api_key_usage:read"))
	apiKeyUsageGroup.GET("", apiKeyUsageController.List)
	apiKeyUsageGroup.GET("/:id", apiKeyUsageController.Get)

	// Campaigns with team-specific permissions
	campaignService := services.NewBaseService(db, models.Campaign{})
	campaignController := controllers.NewBaseController(campaignService)
	campaignGroup := g.Group("/campaigns")
	campaignGroup.Use(middleware.RequirePermissions(db, "campaigns:read"))
	campaignGroup.GET("", campaignController.List)
	campaignGroup.GET("/:id", campaignController.Get)

	// Protected campaign routes
	campaignWriteGroup := campaignGroup.Group("")
	campaignWriteGroup.Use(middleware.RequirePermissions(db, "campaigns:write"))
	campaignWriteGroup.POST("", campaignController.Create)
	campaignWriteGroup.PUT("/:id", campaignController.Update)
	campaignWriteGroup.DELETE("/:id", campaignController.Delete)

	// Automation routes with team-specific permissions
	automationService := services.NewBaseService(db, models.Automation{})
	automationController := controllers.NewBaseController(automationService)
	automationGroup := g.Group("/automations")
	automationGroup.Use(middleware.RequirePermissions(db, "automations:read"))
	automationGroup.GET("", automationController.List)
	automationGroup.GET("/:id", automationController.Get)

	// Protected automation routes
	automationWriteGroup := automationGroup.Group("")
	automationWriteGroup.Use(middleware.RequirePermissions(db, "automations:write"))
	automationWriteGroup.POST("", automationController.Create)
	automationWriteGroup.PUT("/:id", automationController.Update)
	automationWriteGroup.DELETE("/:id", automationController.Delete)

	// Model routes with team-specific permissions
	modelService := services.NewBaseService(db, models.Model{})
	modelController := controllers.NewBaseController(modelService)
	modelGroup := g.Group("/models")
	modelGroup.Use(middleware.RequirePermissions(db, "models:read"))
	modelGroup.GET("", modelController.List)
	modelGroup.GET("/:id", modelController.Get)

	// Protected model routes
	modelWriteGroup := modelGroup.Group("")
	modelWriteGroup.Use(middleware.RequirePermissions(db, "models:write"))
	modelWriteGroup.POST("", modelController.Create)
	modelWriteGroup.PUT("/:id", modelController.Update)
	modelWriteGroup.DELETE("/:id", modelController.Delete)
}
