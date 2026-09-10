package routes

import (
	"github.com/labstack/echo/v4"
	em "github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
	"kori/internal/ai"
	"kori/internal/api/middleware"
	"kori/internal/config"
	"kori/internal/handlers"
	"os"
)

func SetupMarketingRoutes(e *echo.Echo, db *gorm.DB, cfg *config.Config) {
	h := handlers.NewMarketingHandler(db, cfg.JWT.Secret, os.Getenv("PUBLIC_API_URL"))
	g := e.Group("/api/v1/marketing", middleware.NewAuthMiddleware(cfg.JWT.Secret).Middleware())
	endpoint := os.Getenv("AI_PROXY_BASE_URL")
	if endpoint == "" {
		endpoint = "https://ai-proxy.synehq.com/v1"
	}
	model := os.Getenv("AI_PROXY_MODEL")
	if model == "" {
		model = "gpt-4.1"
	}
	writer := ai.NewEmailWriter(endpoint, os.Getenv("AI_PROXY_API_KEY"), model)
	g.POST("/email-draft", handlers.EmailDraftHandler(writer), em.BodyLimit("32K"), em.RateLimiterWithConfig(em.RateLimiterConfig{
		Store:               em.NewRateLimiterMemoryStoreWithConfig(em.RateLimiterMemoryStoreConfig{Rate: rate.Limit(0.1), Burst: 3}),
		IdentifierExtractor: func(c echo.Context) (string, error) { id, _ := c.Get("teamID").(string); return id, nil },
	}))
	g.PUT("/profile", h.SaveProfile)
	g.GET("/branding", h.Branding)
	g.PUT("/branding", h.Branding, middleware.RequirePermissions(db, "branding_settings:update"))
	g.POST("/domains", h.AddDomain, middleware.RequirePermissions(db, "domains:create"))
	g.POST("/domains/:id/verify", h.VerifyDomain, middleware.RequirePermissions(db, "domains:update"))
	g.PUT("/webhooks/:id/status", h.WebhookStatus)
	g.GET("/webhooks/:id/deliveries", h.WebhookDeliveries)
	g.GET("/options", h.Options)
	g.GET("/contacts", h.CRMContacts, middleware.RequirePermissions(db, "contacts:read"))
	g.GET("/tags", h.Tags)
	g.POST("/tags", h.SaveTag)
	g.PUT("/tags/:id", h.SaveTag)
	g.DELETE("/tags/:id", h.DeleteTag)
	g.GET("/contacts/:id/tags", h.ContactTags)
	g.PUT("/contacts/:id/tags", h.ContactTags)
	g.GET("/templates/:id/preview", h.TemplatePreview)
	g.PUT("/contacts/:id/stage", h.UpdateContactStage)
	g.GET("/newsletters", h.Newsletters)
	g.POST("/newsletters", h.SaveNewsletter)
	g.PUT("/newsletters/:id", h.SaveNewsletter)
	g.POST("/newsletters/:id/pause", h.PauseNewsletter)
	g.GET("/newsletters/:id/editions", h.NewsletterEditions)
	g.POST("/template-starters", h.ImportTemplateStarter)
	g.GET("/forms", h.Forms)
	g.POST("/forms", h.SaveForm)
	g.PUT("/forms/:id", h.SaveForm)
	g.GET("/forms/:id/submissions", h.Submissions)
	g.GET("/contacts/:id/notes", h.Notes)
	g.POST("/contacts/:id/notes", h.Notes)
	public := e.Group("/public", em.BodyLimit("32K"), em.RateLimiter(em.NewRateLimiterMemoryStore(rate.Limit(2))))
	public.GET("/forms/:slug", h.PublicForm)
	public.POST("/forms/:slug", h.SubmitForm)
	public.GET("/unsubscribe/:team/:contact/:token", h.Unsubscribe)
	public.POST("/unsubscribe/:team/:contact/:token", h.Unsubscribe)
}
