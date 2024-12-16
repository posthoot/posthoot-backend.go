package registry

import (
	"github.com/labstack/echo/v4"

	"kori/internal/api/controllers"
	"kori/internal/db"
	"kori/internal/models"
	"kori/internal/services"
)

// RegisterCRUDRoutes registers CRUD routes for all models
func RegisterCRUDRoutes(g *echo.Group) {
	// Teams
	teamService := services.NewBaseService(db.DB, models.Team{})
	teamController := controllers.NewBaseController(teamService)
	teamController.RegisterRoutes(g, "/teams", "read", "update")

	// Contacts
	contactService := services.NewBaseService(db.DB, models.Contact{})
	contactController := controllers.NewBaseController(contactService)
	contactController.RegisterRoutes(g, "/contacts")

	// Mailing Lists
	mailingListService := services.NewBaseService(db.DB, models.MailingList{})
	mailingListController := controllers.NewBaseController(mailingListService)
	mailingListController.RegisterRoutes(g, "/mailing-lists")

	// SMTP Configs
	smtpConfigService := services.NewBaseService(db.DB, models.SMTPConfig{})
	smtpConfigController := controllers.NewBaseController(smtpConfigService)
	smtpConfigController.RegisterRoutes(g, "/smtp-configs")

	// Domains
	domainService := services.NewBaseService(db.DB, models.Domain{})
	domainController := controllers.NewBaseController(domainService)
	domainController.RegisterRoutes(g, "/domains")

	// Webhooks
	webhookService := services.NewBaseService(db.DB, models.Webhook{})
	webhookController := controllers.NewBaseController(webhookService)
	webhookController.RegisterRoutes(g, "/webhooks")

	// Templates
	templateService := services.NewBaseService(db.DB, models.Template{})
	templateController := controllers.NewBaseController(templateService)
	templateController.RegisterRoutes(g, "/templates")

	// Emails
	emailService := services.NewBaseService(db.DB, models.Email{})
	emailController := controllers.NewBaseController(emailService)
	emailController.RegisterRoutes(g, "/emails")

	// API Keys
	apiKeyService := services.NewBaseService(db.DB, models.APIKey{})
	apiKeyController := controllers.NewBaseController(apiKeyService)
	apiKeyController.RegisterRoutes(g, "/api-keys")

	// Team Invites
	teamInviteService := services.NewBaseService(db.DB, models.TeamInvite{})
	teamInviteController := controllers.NewBaseController(teamInviteService)
	teamInviteController.RegisterRoutes(g, "/team-invites")

	// Automations
	automationService := services.NewBaseService(db.DB, models.Automation{})
	automationController := controllers.NewBaseController(automationService)
	automationController.RegisterRoutes(g, "/automations")

	// Automation Nodes
	automationNodeService := services.NewBaseService(db.DB, models.AutomationNode{})
	automationNodeController := controllers.NewBaseController(automationNodeService)
	automationNodeController.RegisterRoutes(g, "/automation-nodes")

	// Automation Node Connections
	automationNodeConnectionService := services.NewBaseService(db.DB, models.AutomationNodeEdge{})
	automationNodeConnectionController := controllers.NewBaseController(automationNodeConnectionService)
	automationNodeConnectionController.RegisterRoutes(g, "/automation-node-edges")

	// Models
	modelService := services.NewBaseService(db.DB, models.Model{})
	modelController := controllers.NewBaseController(modelService)
	modelController.RegisterRoutes(g, "/models")

	// Campaigns
	campaignService := services.NewBaseService(db.DB, models.Campaign{})
	campaignController := controllers.NewBaseController(campaignService)
	campaignController.RegisterRoutes(g, "/campaigns")
}
