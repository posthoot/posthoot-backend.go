package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"kori/internal/events"
	"kori/internal/marketing"
	"kori/internal/models"
)

type MarketingHandler struct{ service *marketing.Service }

func NewMarketingHandler(db *gorm.DB, secret, url string) *MarketingHandler {
	return &MarketingHandler{marketing.New(db, secret, url)}
}
func team(c echo.Context) string { v, _ := c.Get("teamID").(string); return v }
func (h *MarketingHandler) scoped(c echo.Context) *gorm.DB {
	return marketing.Scope(h.service.DB.WithContext(c.Request().Context()), team(c))
}
func bad(message string) error { return echo.NewHTTPError(400, message) }
func missing() error           { return echo.NewHTTPError(404, "Not found") }
func (h *MarketingHandler) Options(c echo.Context) error {
	var lists []models.MailingList
	var templates []models.Template
	if err := h.scoped(c).Select("id", "name", "subscribers_count").Find(&lists).Error; err != nil {
		return err
	}
	if err := h.scoped(c).Select("id", "name", "subject", "html_body", "html_file_id", "starter_key").Find(&templates).Error; err != nil {
		return err
	}
	var senders []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		FromEmail   string `json:"fromEmail"`
		SupportsTLS bool   `json:"supportsTLS"`
	}
	if err := h.scoped(c).Model(&models.SMTPConfig{}).Select("id", "provider AS name", "from_email", "supports_tls").Find(&senders).Error; err != nil {
		return err
	}
	return c.JSON(200, map[string]interface{}{"lists": lists, "templates": templates, "senders": senders, "starters": marketing.Starters()})
}
func (h *MarketingHandler) Newsletters(c echo.Context) error {
	var rows []models.Newsletter
	if err := h.scoped(c).Order("created_at DESC").Limit(200).Find(&rows).Error; err != nil {
		return err
	}
	return c.JSON(200, rows)
}
func (h *MarketingHandler) SaveNewsletter(c echo.Context) error {
	var input struct {
		Name, Subject, Description, TemplateID, ListID, SMTPConfigID, Status, Cadence, Timezone, PostalAddress string
		NextSendAt                                                                                             *time.Time
	}
	if err := c.Bind(&input); err != nil {
		return bad("Invalid newsletter")
	}
	n := models.Newsletter{TeamID: team(c), Name: input.Name, Subject: input.Subject, Description: input.Description, TemplateID: input.TemplateID, ListID: input.ListID, SMTPConfigID: input.SMTPConfigID, Status: input.Status, Cadence: input.Cadence, Timezone: input.Timezone, NextSendAt: input.NextSendAt, PostalAddress: input.PostalAddress}
	if err := h.service.ValidateNewsletter(c.Request().Context(), &n); err != nil {
		return bad(err.Error())
	}
	if n.NextSendAt != nil {
		loc, _ := time.LoadLocation(n.Timezone) // Validated above.
		n.MonthDay = n.NextSendAt.In(loc).Day()
	}
	err := h.service.DB.WithContext(c.Request().Context()).Transaction(func(tx *gorm.DB) error {
		if id := c.Param("id"); id != "" {
			var old models.Newsletter
			if err := marketing.Scope(tx, team(c)).Clauses(clause.Locking{Strength: "UPDATE"}).First(&old, "id = ?", id).Error; err != nil {
				return missing()
			}
			n.Base, n.Editions, n.LastSentAt = old.Base, old.Editions, old.LastSentAt
			if n.NextSendAt != nil && old.NextSendAt != nil && n.NextSendAt.Equal(*old.NextSendAt) && n.Timezone == old.Timezone {
				n.MonthDay = old.MonthDay
			}
		}
		return tx.Omit(clause.Associations).Save(&n).Error
	})
	if err != nil {
		return err
	}
	return c.JSON(200, n)
}
func (h *MarketingHandler) PauseNewsletter(c echo.Context) error {
	result := h.scoped(c).Model(&models.Newsletter{}).Where("id = ?", c.Param("id")).Update("status", "PAUSED")
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return missing()
	}
	return c.JSON(200, map[string]string{"status": "PAUSED"})
}
func (h *MarketingHandler) NewsletterEditions(c echo.Context) error {
	if !h.service.Owns(c.Request().Context(), &models.Newsletter{}, c.Param("id"), team(c)) {
		return missing()
	}
	var rows []struct {
		ID           string    `json:"id"`
		Name         string    `json:"name"`
		ScheduledFor time.Time `json:"scheduledFor"`
		Status       string    `json:"status"`
		Processed    int       `json:"processed"`
		Sent         int64     `json:"sent"`
		Failed       int64     `json:"failed"`
		NeedsReview  int64     `json:"needsReview"`
	}
	err := h.service.DB.Table("campaigns c").Select("c.id,c.name,c.scheduled_for,c.status,c.processed,(SELECT count(*) FROM emails e WHERE e.campaign_id=c.id AND e.status='SENT') as sent,(SELECT count(*) FROM emails e WHERE e.campaign_id=c.id AND e.status='FAILED') as failed,(SELECT count(*) FROM emails e WHERE e.campaign_id=c.id AND e.status IN ('DELIVERY_UNKNOWN','SENDING')) as needs_review").Where("c.newsletter_id = ? AND c.team_id = ?", c.Param("id"), team(c)).Order("c.created_at DESC").Limit(50).Scan(&rows).Error
	if err != nil {
		return err
	}
	return c.JSON(200, rows)
}

// ImportTemplateStarter adds a predefined design to the existing template library.
func (h *MarketingHandler) ImportTemplateStarter(c echo.Context) error {
	var input struct {
		StarterKey string `json:"starterKey"`
	}
	if err := c.Bind(&input); err != nil {
		return bad("Choose a starter template")
	}
	var starter *marketing.Starter
	for _, candidate := range marketing.Starters() {
		if candidate.Key == input.StarterKey {
			starter = &candidate
			break
		}
	}
	if starter == nil {
		return bad("Unknown starter template")
	}
	var template models.Template
	err := h.service.DB.Transaction(func(tx *gorm.DB) error {
		var category models.EmailCategory
		if err := marketing.Scope(tx, team(c)).Where("name = ?", "Marketing").FirstOrCreate(&category, models.EmailCategory{Name: "Marketing", TeamID: team(c)}).Error; err != nil {
			return err
		}
		template = models.Template{TeamID: team(c), CategoryID: category.ID, Name: starter.Name, Subject: starter.Subject, HTMLBody: starter.HTMLBody, StarterKey: starter.Key, DesignJSON: starter.DesignJSON}
		return tx.Omit(clause.Associations).Create(&template).Error
	})
	if err != nil {
		return err
	}
	return c.JSON(201, template)
}

type formInput struct {
	Theme          *formTheme `json:"theme"`
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	ListID         string     `json:"listId"`
	Status         string     `json:"status"`
	SuccessMessage string     `json:"successMessage"`
	ButtonText     string     `json:"buttonText"`
	Fields         []struct {
		Label    string `json:"label"`
		Type     string `json:"type"`
		Required bool   `json:"required"`
		Key      string `json:"key"`
	} `json:"fields"`
}

func (h *MarketingHandler) Forms(c echo.Context) error {
	var rows []models.Form
	if err := h.scoped(c).Preload("Fields", func(db *gorm.DB) *gorm.DB { return db.Order("display_order") }).Order("created_at DESC").Limit(200).Find(&rows).Error; err != nil {
		return err
	}
	return c.JSON(200, rows)
}
func (h *MarketingHandler) SaveForm(c echo.Context) error {
	var in formInput
	if err := c.Bind(&in); err != nil {
		return bad("Invalid form")
	}
	if len(strings.TrimSpace(in.Name)) < 2 || len(in.Name) > 120 || len(in.Description) > 500 || len(in.SuccessMessage) > 500 || len(in.ButtonText) > 60 || len(in.Fields) < 1 || len(in.Fields) > 20 {
		return bad("Provide a name and between 1 and 20 fields")
	}
	if in.Status != "DRAFT" && in.Status != "PUBLISHED" && in.Status != "ARCHIVED" {
		return bad("Invalid form status")
	}
	if !h.service.Owns(c.Request().Context(), &models.MailingList{}, in.ListID, team(c)) {
		return bad("Choose an audience in this workspace")
	}
	f := models.Form{TeamID: team(c), FormType: models.FormTypeInline, Slug: uuid.NewString(), Theme: datatypes.JSON(`{}`)}
	if id := c.Param("id"); id != "" {
		if err := h.scoped(c).First(&f, "id = ?", id).Error; err != nil {
			return missing()
		}
	}
	if in.Theme != nil {
		if err := in.Theme.validate(); err != nil {
			return err
		}
		encoded, err := json.Marshal(in.Theme)
		if err != nil {
			return err
		}
		f.Theme = encoded
	}
	f.Name = in.Name
	f.Description = in.Description
	f.AddToListID = &in.ListID
	f.Status = models.FormStatus(in.Status)
	f.SuccessMessage = in.SuccessMessage
	f.SubmitButtonText = in.ButtonText
	fields := []models.FormField{}
	hasEmail := false
	keys := map[string]bool{}
	for i, field := range in.Fields {
		if field.Key == "" || keys[field.Key] || len(field.Label) < 1 || len(field.Label) > 120 {
			return bad("Each field needs a unique key and label")
		}
		keys[field.Key] = true
		switch field.Key {
		case "email", "first_name", "last_name", "company", "phone", "message":
		default:
			return bad("Unsupported field key")
		}
		switch field.Type {
		case "TEXT", "EMAIL", "TEXTAREA", "PHONE":
		default:
			return bad("Unsupported field type")
		}
		if field.Key == "email" {
			if field.Type != "EMAIL" || !field.Required {
				return bad("Email must be a required email field")
			}
			hasEmail = true
		}
		key := field.Key
		fields = append(fields, models.FormField{FieldType: models.FieldType(field.Type), Label: field.Label, Required: field.Required, DisplayOrder: i, MapToContactField: &key, Options: datatypes.JSON(`[]`), ConditionalDisplay: datatypes.JSON(`{}`)})
	}
	if !hasEmail {
		return bad("A required email field is needed")
	}
	err := h.service.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit(clause.Associations).Save(&f).Error; err != nil {
			return err
		}
		if err := tx.Where("form_id = ?", f.ID).Delete(&models.FormField{}).Error; err != nil {
			return err
		}
		for i := range fields {
			fields[i].FormID = f.ID
		}
		return tx.Create(&fields).Error
	})
	if err != nil {
		return err
	}
	f.Fields = fields
	return c.JSON(200, f)
}
func (h *MarketingHandler) Submissions(c echo.Context) error {
	if !h.service.Owns(c.Request().Context(), &models.Form{}, c.Param("id"), team(c)) {
		return missing()
	}
	var rows []models.FormSubmission
	if err := h.service.DB.Where("form_id = ?", c.Param("id")).Select("id", "email_address", "field_data", "submitted_at", "contact_id").Order("submitted_at DESC").Limit(200).Find(&rows).Error; err != nil {
		return err
	}
	return c.JSON(200, rows)
}
func (h *MarketingHandler) PublicForm(c echo.Context) error {
	var f models.Form
	if err := h.service.DB.Preload("Fields", func(db *gorm.DB) *gorm.DB { return db.Order("display_order") }).Where("slug = ? AND status = ? AND is_deleted = ?", c.Param("slug"), models.FormStatusPublished, false).First(&f).Error; err != nil {
		return missing()
	}
	if err := h.service.DB.Model(&f).UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error; err != nil {
		return err
	}
	// Explicit public projection: never serialize internal routing or submission data.
	return c.JSON(200, map[string]interface{}{"name": f.Name, "description": f.Description, "fields": f.Fields, "buttonText": f.SubmitButtonText, "successMessage": f.SuccessMessage, "theme": f.Theme})
}
func (h *MarketingHandler) submitForm(c echo.Context) error {
	in, err := bindFormSubmission(c)
	if err != nil {
		return bad("Invalid submission")
	}
	if in.Website != "" {
		return formSubmissionSuccess(c, "Thanks for joining us!")
	}
	if !in.Consent || uuid.Validate(in.RequestID) != nil {
		return bad("Consent and a submission identifier are required")
	}
	var f models.Form
	if err := h.service.DB.Preload("Fields").Where("slug = ? AND status = ? AND is_deleted = ?", c.Param("slug"), models.FormStatusPublished, false).First(&f).Error; err != nil {
		return missing()
	}
	data := map[string]string{}
	for _, field := range f.Fields {
		if field.MapToContactField == nil {
			continue
		}
		key := *field.MapToContactField
		v := strings.TrimSpace(in.Fields[key])
		if (field.Required && v == "") || len(v) > 2000 {
			return bad("Complete the required fields (maximum 2,000 characters each)")
		}
		data[key] = v
	}
	email := strings.ToLower(data["email"])
	if !marketing.ValidEmail(email) {
		return bad("Enter a valid email address")
	}
	if f.AddToListID == nil || !h.service.Owns(c.Request().Context(), &models.MailingList{}, *f.AddToListID, f.TeamID) {
		return bad("This form is unavailable")
	}
	encoded, _ := json.Marshal(data)
	var contact models.Contact
	created := false
	sum := sha256.Sum256([]byte(f.ID + ":" + in.RequestID))
	receipt := models.FormReceipt{ID: hex.EncodeToString(sum[:]), FormID: f.ID}
	err = h.service.DB.Transaction(func(tx *gorm.DB) error {
		// Serialize edits/capture, then lock the audience shared by different forms.
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&f, "id = ?", f.ID).Error; err != nil {
			return err
		}
		if f.Status != models.FormStatusPublished || f.IsDeleted || f.AddToListID == nil {
			return bad("This form is unavailable")
		}
		var audience models.MailingList
		if err := marketing.Scope(tx, f.TeamID).Clauses(clause.Locking{Strength: "UPDATE"}).First(&audience, "id = ?", *f.AddToListID).Error; err != nil {
			return bad("This form is unavailable")
		}
		r := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&receipt)
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return nil
		}
		err := marketing.Scope(tx, f.TeamID).Where("email = ? AND list_id = ?", email, *f.AddToListID).First(&contact).Error
		if err == gorm.ErrRecordNotFound {
			created = true
			contact = models.Contact{Base: models.Base{ID: uuid.NewString(), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}, TeamID: f.TeamID, ListID: *f.AddToListID, Email: email, FirstName: data["first_name"], LastName: data["last_name"], Company: data["company"], Phone: data["phone"], Status: models.SubscriberStatusActive, LifecycleStage: "LEAD", Metadata: datatypes.JSON(`{"source":"form","consent":true}`)}
			if err := tx.Session(&gorm.Session{SkipHooks: true}).Create(&contact).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		// Existing suppressed contacts retain their status and are never silently resubscribed.
		sub := models.FormSubmission{FormID: f.ID, ContactID: &contact.ID, EmailAddress: email, FieldData: encoded, SubmittedAt: time.Now()}
		if err := tx.Create(&sub).Error; err != nil {
			return err
		}
		if err := models.SyncSubscribersCountByID(tx, *f.AddToListID); err != nil {
			return err
		}
		return tx.Model(&f).UpdateColumn("submission_count", gorm.Expr("submission_count + 1")).Error
	})
	if err != nil {
		return err
	}
	if created {
		events.Emit("contact.created", &contact)
	}
	return formSubmissionSuccess(c, f.SuccessMessage)
}
func (h *MarketingHandler) Unsubscribe(c echo.Context) error {
	messageID := c.QueryParam("message")
	valid := h.service.CheckToken(c.Param("team"), c.Param("contact"), c.Param("token"))
	if messageID != "" {
		valid = h.service.CheckMessageToken(c.Param("team"), c.Param("contact"), messageID, c.Param("token"))
	}
	if !valid {
		return missing()
	}
	if c.Request().Method == http.MethodGet {
		c.Response().Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; base-uri 'none'")
		return c.HTML(200, `<!doctype html><html><meta name="viewport" content="width=device-width"><title>Unsubscribe</title><body style="font:16px Arial;max-width:480px;margin:80px auto;padding:24px"><h1>Unsubscribe from this audience</h1><p>You can stop receiving these emails at any time.</p><form method="post"><button style="padding:12px 24px">Confirm unsubscribe</button></form></body></html>`)
	}
	var contact models.Contact
	err := h.service.DB.Transaction(func(tx *gorm.DB) error {
		if err := marketing.Scope(tx, c.Param("team")).Clauses(clause.Locking{Strength: "UPDATE"}).First(&contact, "id = ?", c.Param("contact")).Error; err != nil {
			return err
		}
		if messageID != "" && contact.Status != models.SubscriberStatusUnsubscribed {
			var message models.Email
			if err := tx.Where("id=? AND team_id=? AND contact_id=?", messageID, contact.TeamID, contact.ID).First(&message).Error; err != nil {
				return err
			}
			if err := tx.Create(&models.EmailTracking{EmailID: message.ID, CampaignID: message.CampaignID, ContactID: contact.ID, Event: models.EmailTrackingEventUnsubscribe, Timestamp: time.Now().UTC()}).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&contact).UpdateColumn("status", models.SubscriberStatusUnsubscribed).Error; err != nil {
			return err
		}
		return models.SyncSubscribersCountByID(tx, contact.ListID)
	})
	if err != nil {
		return missing()
	}
	return c.HTML(200, "<p>You have been unsubscribed.</p>")
}
func (h *MarketingHandler) Notes(c echo.Context) error {
	id := c.Param("id")
	if !h.service.Owns(c.Request().Context(), &models.Contact{}, id, team(c)) {
		return missing()
	}
	if c.Request().Method == "POST" {
		var in struct{ Body string }
		if err := c.Bind(&in); err != nil || len(strings.TrimSpace(in.Body)) == 0 || len(in.Body) > 5000 {
			return bad("A note of up to 5,000 characters is required")
		}
		author, _ := c.Get("userID").(string)
		n := models.ContactNote{TeamID: team(c), ContactID: id, Body: strings.TrimSpace(in.Body), AuthorID: author}
		if err := h.service.DB.Create(&n).Error; err != nil {
			return err
		}
		return c.JSON(201, n)
	}
	var notes []models.ContactNote
	if err := h.scoped(c).Where("contact_id = ?", id).Order("created_at DESC").Limit(100).Find(&notes).Error; err != nil {
		return err
	}
	return c.JSON(200, notes)
}

func (h *MarketingHandler) TemplatePreview(c echo.Context) error {
	var template models.Template
	if err := h.scoped(c).First(&template, "id = ?", c.Param("id")).Error; err != nil {
		return missing()
	}
	body, err := marketing.TemplateHTML(h.service.DB.WithContext(c.Request().Context()), &template)
	if err != nil {
		return bad("Template preview unavailable; open the template editor to check its content")
	}
	return c.JSON(200, map[string]string{"htmlBody": body})
}

// UpdateContactStage extends existing contacts without replacing contact CRUD.
func (h *MarketingHandler) UpdateContactStage(c echo.Context) error {
	var in struct {
		LifecycleStage string `json:"lifecycleStage"`
	}
	if err := c.Bind(&in); err != nil {
		return bad("Choose a lifecycle stage")
	}
	switch in.LifecycleStage {
	case "LEAD", "QUALIFIED", "CUSTOMER", "LOST":
	default:
		return bad("Choose a lifecycle stage")
	}
	result := h.scoped(c).Model(&models.Contact{}).Where("id = ?", c.Param("id")).UpdateColumn("lifecycle_stage", in.LifecycleStage)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return missing()
	}
	return c.JSON(200, map[string]string{"lifecycleStage": in.LifecycleStage})
}
