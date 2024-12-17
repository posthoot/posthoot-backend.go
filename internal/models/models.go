package models

import (
	"encoding/json"
	"kori/internal/events"
	"time"

	"gorm.io/gorm"
)

type Team struct {
	Base
	Name            string          `gorm:"not null" json:"name"`
	Settings        *TeamSettings   `gorm:"type:jsonb" json:"settings"`
	Users           []User          `gorm:"foreignKey:TeamID" json:"users,omitempty"`
	MailingLists    []MailingList   `gorm:"foreignKey:TeamID" json:"mailingLists,omitempty"`
	Webhooks        []Webhook       `gorm:"foreignKey:TeamID" json:"webhooks,omitempty"`
	SMTPConfigs     []SMTPConfig    `gorm:"foreignKey:TeamID" json:"smtpConfigs,omitempty"`
	Domains         []Domain        `gorm:"foreignKey:TeamID" json:"domains,omitempty"`
	Invites         []TeamInvite    `gorm:"foreignKey:TeamID" json:"invites,omitempty"`
	APIKeys         []APIKey        `gorm:"foreignKey:TeamID" json:"apiKeys,omitempty"`
	Automations     []Automation    `gorm:"foreignKey:TeamID" json:"automations,omitempty"`
	Models          []Model         `gorm:"foreignKey:TeamID" json:"models,omitempty"`
	EmailCategories []EmailCategory `gorm:"foreignKey:TeamID" json:"emailCategories,omitempty"`
	Campaigns       []Campaign      `gorm:"foreignKey:TeamID" json:"campaigns,omitempty"`
}

func (t *Team) AfterCreate(tx *gorm.DB) error {
	// First load initial data
	if err := LoadInitialData(tx, t.ID); err != nil {
		return err
	}

	// Emit team created event
	events.Emit("team.created", t)
	return nil
}

type TeamInvite struct {
	Base
	Email     string       `gorm:"not null" json:"email"`
	Name      string       `gorm:"not null" json:"name"`
	TeamID    string       `gorm:"type:uuid;not null" json:"teamId"`
	Team      *Team        `json:"team,omitempty"`
	InviterID string       `gorm:"type:uuid;not null" json:"inviterId"`
	Inviter   *User        `json:"inviter,omitempty"`
	Status    InviteStatus `gorm:"not null;default:'PENDING'" json:"status"`
	ExpiresAt time.Time    `gorm:"not null" json:"expiresAt"`
}

type Contact struct {
	Base
	Email     string                 `gorm:"not null" json:"email"`
	FirstName string                 `json:"firstName"`
	LastName  string                 `json:"lastName"`
	Metadata  map[string]interface{} `gorm:"type:jsonb" json:"metadata"`
	Tags      []Tag                  `gorm:"many2many:contact_tags;" json:"tags"`
	ListID    string                 `gorm:"type:uuid;not null" json:"listId"`
	TeamID    string                 `gorm:"type:uuid;not null" json:"teamId"`
	List      *MailingList           `json:"list,omitempty"`
	ImportID  string                 `gorm:"type:uuid;not null" json:"importId"`
	Import    *ContactImport         `json:"import,omitempty"`
}

type ContactImport struct {
	Base
	Status    ContactImportStatus `gorm:"not null;default:'PENDING'" json:"status"`
	TeamID    string              `gorm:"type:uuid;not null" json:"teamId"`
	Team      *Team               `json:"team,omitempty"`
	FileID    string              `gorm:"type:uuid;not null" json:"fileId"`
	File      *File               `json:"file,omitempty"`
	FieldsMap string              `gorm:"type:jsonb" json:"fieldsMap"`
	Contacts  []Contact           `gorm:"foreignKey:ImportID" json:"contacts,omitempty"`
}

type File struct {
	Base
	TeamID string `gorm:"type:uuid;not null" json:"teamId"`
	Team   *Team  `json:"team,omitempty"`
	Path   string `gorm:"not null" json:"path"`
}

type Tag struct {
	Base
	Name  string `gorm:"not null" json:"name"`
	Value string `json:"value"`
}

type MailingList struct {
	Base
	Name        string    `gorm:"not null" json:"name"`
	Description string    `json:"description"`
	TeamID      string    `gorm:"type:uuid;not null" json:"teamId"`
	Team        *Team     `json:"team,omitempty"`
	Contacts    []Contact `gorm:"foreignKey:ListID" json:"contacts,omitempty"`
}

type SMTPConfig struct {
	Base
	Provider     string `gorm:"not null" json:"provider"`
	Host         string `gorm:"not null" json:"host"`
	Port         int    `gorm:"not null" json:"port"`
	Username     string `json:"username"`
	Password     string `json:"-"`
	IsDefault    bool   `gorm:"not null;default:false" json:"isDefault"`
	IsActive     bool   `gorm:"not null;default:false" json:"isActive"`
	SupportsTLS  bool   `gorm:"not null;default:true" json:"supportsTls"`
	RequiresAuth bool   `gorm:"not null;default:true" json:"requiresAuth"`
	MaxSendRate  int    `gorm:"not null;default:10" json:"maxSendRate"`
	TeamID       string `gorm:"type:uuid;not null" json:"teamId"`
}

type Domain struct {
	Base
	Domain     string `gorm:"uniqueIndex;not null" json:"domain"`
	IsVerified bool   `gorm:"not null;default:false" json:"isVerified"`
	DNSRecord  string `json:"dnsRecord"`
	TeamID     string `gorm:"type:uuid;not null" json:"teamId"`
}

type Webhook struct {
	Base
	Name       string      `gorm:"not null" json:"name"`
	URL        string      `gorm:"not null" json:"url"`
	Events     StringArray `gorm:"type:text[]" json:"events"`
	IsActive   bool        `gorm:"not null;default:true" json:"isActive"`
	Secret     string      `json:"-"`
	TeamID     string      `gorm:"type:uuid;not null" json:"teamId"`
	Deliveries []Delivery  `gorm:"foreignKey:WebhookID" json:"deliveries,omitempty"`
}

type Delivery struct {
	Base
	WebhookID    string          `gorm:"type:uuid;not null" json:"webhookId"`
	Event        string          `gorm:"not null" json:"event"`
	Payload      json.RawMessage `gorm:"type:jsonb;not null" json:"payload"`
	ResponseCode int             `json:"responseCode"`
	ResponseBody string          `json:"responseBody"`
	Error        string          `json:"error"`
	Status       string          `gorm:"not null" json:"status"`
}

type EmailCategory struct {
	Base
	Name        string     `gorm:"not null" json:"name"`
	Description string     `json:"description"`
	Emails      []Email    `gorm:"foreignKey:CategoryID" json:"emails,omitempty"`
	Templates   []Template `gorm:"foreignKey:CategoryID" json:"templates,omitempty"`
	TeamID      string     `gorm:"type:uuid;not null" json:"teamId"`
	Team        *Team      `json:"team,omitempty"`
}

type Template struct {
	Base
	Name       string         `gorm:"not null" json:"name"`
	Subject    string         `gorm:"not null" json:"subject"`
	HtmlFile   string         `gorm:"not null" json:"htmlFile"`
	DesignJSON string         `gorm:"not null" json:"designJson"`
	TeamID     string         `gorm:"type:uuid;not null" json:"teamId"`
	Team       *Team          `json:"team,omitempty"`
	Emails     []Email        `gorm:"foreignKey:TemplateID" json:"emails,omitempty"`
	Variables  StringArray    `gorm:"type:text[]" json:"variables"`
	CategoryID string         `gorm:"type:uuid;not null" json:"categoryId"`
	Category   *EmailCategory `json:"category,omitempty"`
}

type Email struct {
	Base
	From         string          `gorm:"not null" json:"from"`
	To           string          `gorm:"not null" json:"to"`
	Subject      string          `gorm:"not null" json:"subject"`
	Body         string          `gorm:"not null" json:"body"`
	Status       EmailStatus     `gorm:"not null" json:"status"`
	Error        string          `json:"error"`
	Data         json.RawMessage `gorm:"type:jsonb" json:"data"`
	TemplateID   string          `gorm:"type:uuid;not null" json:"templateId"`
	Template     *Template       `json:"template,omitempty"`
	TeamID       string          `gorm:"type:uuid;not null" json:"teamId"`
	Team         *Team           `json:"team,omitempty"`
	ContactID    string          `gorm:"type:uuid;not null" json:"contactId"`
	Contact      *Contact        `json:"contact,omitempty"`
	SMTPConfigID string          `gorm:"type:uuid;not null" json:"smtpConfigId"`
	SMTPConfig   *SMTPConfig     `json:"smtpConfig,omitempty"`
	SentAt       time.Time       `json:"sentAt"`
	CategoryID   string          `gorm:"type:uuid;not null" json:"categoryId"`
	Category     *EmailCategory  `json:"category,omitempty"`
	CampaignID   string          `gorm:"type:uuid" json:"campaignId"`
	Campaign     *Campaign       `json:"campaign,omitempty"`
}

type EmailTracking struct {
	Base
	EmailID    string             `gorm:"type:uuid;not null" json:"emailId"`
	Email      *Email             `json:"email,omitempty"`
	CampaignID string             `gorm:"type:uuid;not null" json:"campaignId"`
	Campaign   *Campaign          `json:"campaign,omitempty"`
	ContactID  string             `gorm:"type:uuid;not null" json:"contactId"`
	Contact    *Contact           `json:"contact,omitempty"`
	Event      EmailTrackingEvent `gorm:"not null" json:"event"`
	Timestamp  time.Time          `json:"timestamp"`
}

type EmailClick struct {
	Base
	EmailID   string    `gorm:"type:uuid;not null" json:"emailId"`
	Email     *Email    `json:"email,omitempty"`
	URL       string    `gorm:"not null" json:"url"`
	Timestamp time.Time `json:"timestamp"`
}

type EmailOpen struct {
	Base
	EmailID   string    `gorm:"type:uuid;not null" json:"emailId"`
	Email     *Email    `json:"email,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type EmailReply struct {
	Base
	EmailID   string    `gorm:"type:uuid;not null" json:"emailId"`
	Email     *Email    `json:"email,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type EmailBounce struct {
	Base
	EmailID   string    `gorm:"type:uuid;not null" json:"emailId"`
	Email     *Email    `json:"email,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type EmailComplaint struct {
	Base
	EmailID   string    `gorm:"type:uuid;not null" json:"emailId"`
	Email     *Email    `json:"email,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type APIKey struct {
	Base
	Key         string             `gorm:"not null" json:"key"`
	TeamID      string             `gorm:"type:uuid;not null" json:"teamId"`
	Team        *Team              `json:"team,omitempty"`
	CreatedAt   time.Time          `json:"createdAt"`
	LastUsedAt  time.Time          `json:"lastUsedAt"`
	ExpiresAt   time.Time          `json:"expiresAt"`
	Permissions []APIKeyPermission `gorm:"foreignKey:KeyID" json:"permissions,omitempty"`
}

type APIKeyUsage struct {
	Base
	APIKeyID  string    `gorm:"type:uuid;not null" json:"apiKeyId"`
	APIKey    *APIKey   `json:"apiKey,omitempty"`
	Endpoint  string    `gorm:"not null" json:"endpoint"`
	Method    string    `gorm:"not null" json:"method"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `gorm:"not null;default:true" json:"success"`
	Error     string    `json:"error"`
	IPAddress string    `json:"ipAddress"`
	UserAgent string    `json:"userAgent"`
}

type Campaign struct {
	Base
	Name              string                    `gorm:"not null" json:"name"`
	Description       string                    `json:"description"`
	TemplateID        string                    `gorm:"type:uuid;not null" json:"templateId"`
	Template          *Template                 `json:"template,omitempty"`
	TeamID            string                    `gorm:"type:uuid;not null" json:"teamId"`
	Team              *Team                     `json:"team,omitempty"`
	Status            CampaignStatus            `gorm:"not null;default:'DRAFT'" json:"status"`
	ScheduledFor      time.Time                 `json:"scheduledFor"`
	Schedule          CampaignSchedule          `json:"schedule"`
	ListID            string                    `gorm:"type:uuid;not null" json:"listId"`
	List              *MailingList              `json:"list,omitempty"`
	RecurringSchedule CampaignRecurringSchedule `json:"recurringSchedule"`
	CronExpression    string                    `json:"cronExpression"`
	SentEmails        []Email                   `gorm:"foreignKey:CampaignID" json:"sentEmails,omitempty"`
	Analytics         []EmailTracking           `gorm:"foreignKey:CampaignID" json:"analytics,omitempty"`
	SMTPConfigID      string                    `gorm:"type:uuid;not null" json:"smtpConfigId"`
	SMTPConfig        *SMTPConfig               `json:"smtpConfig,omitempty"`
}
type RateLimit struct {
	Base
	APIKeyID  string        `gorm:"type:uuid;not null" json:"apiKeyId"`
	APIKey    *APIKey       `json:"apiKey,omitempty"`
	Limit     int           `gorm:"not null" json:"limit"`
	Period    time.Duration `gorm:"not null" json:"period"`
	CreatedAt time.Time     `json:"createdAt"`
}

type Automation struct {
	Base
	Name        string               `gorm:"not null" json:"name"`
	Description string               `json:"description"`
	TeamID      string               `gorm:"type:uuid;not null" json:"teamId"`
	Team        *Team                `json:"team,omitempty"`
	Nodes       []AutomationNode     `gorm:"foreignKey:AutomationID" json:"nodes,omitempty"`
	Edges       []AutomationNodeEdge `gorm:"foreignKey:AutomationID" json:"edges,omitempty"`
	IsActive    bool                 `gorm:"not null;default:true" json:"isActive"`
}

type Model struct {
	Base
	Name        string `gorm:"not null" json:"name"`
	Description string `json:"description"`
	TeamID      string `gorm:"type:uuid;not null" json:"teamId"`
	Team        *Team  `json:"team,omitempty"`
	Provider    string `gorm:"not null" json:"provider"`
}

type LLMEmailWriterJob struct {
	Base
	AutomationID string      `gorm:"type:uuid;not null" json:"automationId"`
	Automation   *Automation `json:"automation,omitempty"`
	EmailID      string      `gorm:"type:uuid;not null" json:"emailId"`
	Email        *Email      `json:"email,omitempty"`
	Status       string      `gorm:"not null" json:"status"`
	CreatedAt    time.Time   `json:"createdAt"`
	StartedAt    time.Time   `json:"startedAt"`
	CompletedAt  time.Time   `json:"completedAt"`
	UpdatedAt    time.Time   `json:"updatedAt"`
	Error        string      `json:"error"`
	Output       string      `json:"output"`
	Input        string      `json:"input"`
	Prompt       string      `json:"prompt"`
	Model        *Model      `json:"model,omitempty"`
	ModelID      string      `gorm:"type:uuid;not null" json:"modelId"`
}

type AutomationNode struct {
	Base
	AutomationID string               `gorm:"type:uuid;not null" json:"automationId"`
	Automation   *Automation          `json:"automation,omitempty"`
	Type         NodeType             `gorm:"not null" json:"type"`
	Data         json.RawMessage      `gorm:"type:jsonb" json:"data"`
	EdgesFrom    []AutomationNodeEdge `gorm:"foreignKey:SourceID" json:"edgesFrom,omitempty"`
	EdgesTo      []AutomationNodeEdge `gorm:"foreignKey:TargetID" json:"edgesTo,omitempty"`
}

type AutomationNodeEdge struct {
	Base
	AutomationID string          `gorm:"type:uuid;not null" json:"automationId"`
	Automation   *Automation     `json:"automation,omitempty"`
	SourceID     string          `gorm:"type:uuid;not null" json:"sourceId"`
	Source       *AutomationNode `json:"source,omitempty"`
	TargetID     string          `gorm:"type:uuid;not null" json:"targetId"`
	Target       *AutomationNode `json:"target,omitempty"`
	Label        string          `json:"label"`
	Animated     bool            `gorm:"not null;default:true" json:"animated"`
}
