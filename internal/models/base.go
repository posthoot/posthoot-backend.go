package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base contains common columns for all tables
type Base struct {
	ID        string    `gorm:"type:uuid;primary_key" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	DeletedAt time.Time `gorm:"index" json:"-"`
	IsDeleted bool      `json:"isDeleted"`
}

// BeforeCreate will set a UUID rather than numeric ID
func (base *Base) BeforeCreate(tx *gorm.DB) error {
	if base.ID == "" {
		base.ID = uuid.New().String()
	}
	return nil
}

// Status enums
type CampaignStatus string
type CampaignSchedule string
type CampaignRecurringSchedule string
type EmailStatus string
type JobStatus string
type SubscriberStatus string
type TrackingType string
type SMTPProvider string
type NodeType string

// Campaign status constants
const (
	CampaignStatusDraft     CampaignStatus = "DRAFT"
	CampaignStatusScheduled CampaignStatus = "SCHEDULED"
	CampaignStatusSending   CampaignStatus = "SENDING"
	CampaignStatusCompleted CampaignStatus = "COMPLETED"
	CampaignStatusFailed    CampaignStatus = "FAILED"
	CampaignStatusPaused    CampaignStatus = "PAUSED"
)

// Campaign schedule constants
const (
	CampaignScheduleOneTime   CampaignSchedule = "ONE_TIME"
	CampaignScheduleRecurring CampaignSchedule = "RECURRING"
)

// Campaign recurring schedule constants
const (
	CampaignRecurringScheduleDaily   CampaignRecurringSchedule = "DAILY"
	CampaignRecurringScheduleWeekly  CampaignRecurringSchedule = "WEEKLY"
	CampaignRecurringScheduleMonthly CampaignRecurringSchedule = "MONTHLY"
	CampaignRecurringScheduleCustom  CampaignRecurringSchedule = "CUSTOM"
)

// Email status constants
const (
	EmailStatusPending EmailStatus = "PENDING"
	EmailStatusSent    EmailStatus = "SENT"
	EmailStatusFailed  EmailStatus = "FAILED"
	EmailStatusBounced EmailStatus = "BOUNCED"
	EmailStatusOpened  EmailStatus = "OPENED"
	EmailStatusClicked EmailStatus = "CLICKED"
)

// Job status constants
const (
	JobStatusQueued     JobStatus = "QUEUED"
	JobStatusProcessing JobStatus = "PROCESSING"
	JobStatusCompleted  JobStatus = "COMPLETED"
	JobStatusFailed     JobStatus = "FAILED"
	JobStatusCancelled  JobStatus = "CANCELLED"
)

// Subscriber status constants
const (
	SubscriberStatusActive       SubscriberStatus = "ACTIVE"
	SubscriberStatusUnsubscribed SubscriberStatus = "UNSUBSCRIBED"
	SubscriberStatusBounced      SubscriberStatus = "BOUNCED"
	SubscriberStatusComplained   SubscriberStatus = "COMPLAINED"
)

// Tracking type constants
const (
	TrackingTypeClicked TrackingType = "CLICKED"
	TrackingTypeOpened  TrackingType = "OPENED"
	TrackingTypeBounced TrackingType = "BOUNCED"
	TrackingTypeFailed  TrackingType = "FAILED"
)

// SMTP provider constants
const (
	SMTPProviderCustom  SMTPProvider = "CUSTOM"
	SMTPProviderGmail   SMTPProvider = "GMAIL"
	SMTPProviderOutlook SMTPProvider = "OUTLOOK"
	SMTPProviderAmazon  SMTPProvider = "AMAZON"
)

// Node type constants
const (
	NodeTypeStart            NodeType = "START"
	NodeTypeEmail            NodeType = "EMAIL"
	NodeTypeWait             NodeType = "WAIT"
	NodeTypeCondition        NodeType = "CONDITION"
	NodeTypeWebhook          NodeType = "WEBHOOK"
	NodeTypeAddToList        NodeType = "ADD_TO_LIST"
	NodeTypeRemoveFromList   NodeType = "REMOVE_FROM_LIST"
	NodeTypeUpdateSubscriber NodeType = "UPDATE_SUBSCRIBER"
	NodeTypeCheckEngagement  NodeType = "CHECK_ENGAGEMENT"
	NodeTypeSegment          NodeType = "SEGMENT"
	NodeTypeTag              NodeType = "TAG"
	NodeTypeUnsubscribe      NodeType = "UNSUBSCRIBE"
	NodeTypeCustomCode       NodeType = "CUSTOM_CODE"
	NodeTypeExit             NodeType = "EXIT"
)

type UserRole string

const (
	UserRoleSuperAdmin UserRole = "SUPER_ADMIN"
	UserRoleAdmin      UserRole = "ADMIN"
	UserRoleMember     UserRole = "MEMBER"
)

type InviteStatus string

const (
	InviteStatusPending  InviteStatus = "PENDING"
	InviteStatusAccepted InviteStatus = "ACCEPTED"
	InviteStatusRejected InviteStatus = "REJECTED"
)

type TeamSettings struct {
	InviteTemplateID  string           `json:"inviteTemplateId"`
	WelcomeTemplateID string           `json:"welcomeTemplateId"`
	BrandingSettings  BrandingSettings `json:"branding"`
}

type BrandingSettings struct {
	DashboardName string `json:"dashboardName"`
	LogoURL       string `json:"logoUrl"`
}

type ContactImportStatus string

const (
	ContactImportStatusPending   ContactImportStatus = "PENDING"
	ContactImportStatusCompleted ContactImportStatus = "COMPLETED"
	ContactImportStatusFailed    ContactImportStatus = "FAILED"
)

type EmailTrackingEvent string

const (
	EmailTrackingEventClick     EmailTrackingEvent = "click"
	EmailTrackingEventOpen      EmailTrackingEvent = "open"
	EmailTrackingEventReply     EmailTrackingEvent = "reply"
	EmailTrackingEventBounce    EmailTrackingEvent = "bounce"
	EmailTrackingEventComplaint EmailTrackingEvent = "complaint"
)

// StringArray is a custom type for handling string arrays in PostgreSQL
type StringArray []string

// Scan implements the sql.Scanner interface
func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = StringArray{}
		return nil
	}

	var array []string
	switch v := value.(type) {
	case []byte:
		if err := json.Unmarshal(v, &array); err != nil {
			return err
		}
		*a = StringArray(array)
		return nil
	default:
		return errors.New("failed to scan StringArray")
	}
}

// Value implements the driver.Valuer interface
func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	return json.Marshal(a)
}
