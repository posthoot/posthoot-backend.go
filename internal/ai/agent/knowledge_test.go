package agent

import (
	"context"
	"kori/internal/models"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	// Set test mode to skip seeders
	os.Setenv("TEST_MODE", "true")

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.BrandingSettings{},
		&models.TeamSettings{},
		&models.Team{},
		&models.Contact{},
		&models.Campaign{},
		&models.EmailTracking{},
		&models.Tag{},
	)
	require.NoError(t, err)

	return db
}

func createTestTeam(t *testing.T, db *gorm.DB) *models.Team {
	team := &models.Team{
		Base: models.Base{ID: uuid.New().String()},
		Name: "Test Team",
	}
	require.NoError(t, db.Create(team).Error)
	return team
}

func createTestContact(t *testing.T, db *gorm.DB, teamID string) *models.Contact {
	contact := &models.Contact{
		Base:   models.Base{ID: uuid.New().String()},
		TeamID: teamID,
		Email:  "test@example.com",
		Name:   "Test Contact",
	}
	require.NoError(t, db.Create(contact).Error)
	return contact
}

func createTestCampaign(t *testing.T, db *gorm.DB, teamID string) *models.Campaign {
	campaign := &models.Campaign{
		Base:    models.Base{ID: uuid.New().String()},
		TeamID:  teamID,
		Name:    "Test Campaign",
		Subject: "Test Subject",
		Status:  models.CampaignStatusCompleted,
	}
	require.NoError(t, db.Create(campaign).Error)
	return campaign
}

func TestAnalyticsAggregator_AggregateEmailAnalytics(t *testing.T) {
	db := setupTestDB(t)
	aggregator := NewAnalyticsAggregator(db)

	team := createTestTeam(t, db)
	contact := createTestContact(t, db, team.ID)
	campaign := createTestCampaign(t, db, team.ID)

	// Create email tracking data
	now := time.Now()
	trackings := []models.EmailTracking{
		{
			Base:       models.Base{ID: uuid.New().String()},
			TeamID:     team.ID,
			ContactID:  contact.ID,
			CampaignID: campaign.ID,
			Status:     models.EmailStatusDelivered,
			OpenedAt:   &now,
			ClickedAt:  &now,
		},
		{
			Base:       models.Base{ID: uuid.New().String()},
			TeamID:     team.ID,
			ContactID:  contact.ID,
			CampaignID: campaign.ID,
			Status:     models.EmailStatusDelivered,
			OpenedAt:   &now,
		},
		{
			Base:       models.Base{ID: uuid.New().String()},
			TeamID:     team.ID,
			ContactID:  contact.ID,
			CampaignID: campaign.ID,
			Status:     models.EmailStatusDelivered,
		},
		{
			Base:       models.Base{ID: uuid.New().String()},
			TeamID:     team.ID,
			ContactID:  contact.ID,
			CampaignID: campaign.ID,
			Status:     models.EmailStatusBounced,
		},
	}

	for _, tracking := range trackings {
		require.NoError(t, db.Create(&tracking).Error)
	}

	// Aggregate analytics
	ctx := context.Background()
	analytics, err := aggregator.AggregateEmailAnalytics(ctx, team.ID, 30)
	require.NoError(t, err)

	// Verify results
	assert.Equal(t, int64(4), analytics.TotalSent)
	assert.Equal(t, 50.0, analytics.OpenRate)      // 2/4 * 100
	assert.Equal(t, 25.0, analytics.ClickRate)     // 1/4 * 100
	assert.Equal(t, 25.0, analytics.BounceRate)    // 1/4 * 100
}

func TestAnalyticsAggregator_AggregateCampaignAnalytics(t *testing.T) {
	db := setupTestDB(t)
	aggregator := NewAnalyticsAggregator(db)

	team := createTestTeam(t, db)
	contact := createTestContact(t, db, team.ID)
	campaign := createTestCampaign(t, db, team.ID)

	// Create email tracking with device and time data
	now := time.Now()
	trackings := []models.EmailTracking{
		{
			Base:       models.Base{ID: uuid.New().String()},
			TeamID:     team.ID,
			ContactID:  contact.ID,
			CampaignID: campaign.ID,
			Status:     models.EmailStatusDelivered,
			OpenedAt:   &now,
			Device:     "mobile",
		},
		{
			Base:       models.Base{ID: uuid.New().String()},
			TeamID:     team.ID,
			ContactID:  contact.ID,
			CampaignID: campaign.ID,
			Status:     models.EmailStatusDelivered,
			OpenedAt:   &now,
			Device:     "desktop",
		},
		{
			Base:       models.Base{ID: uuid.New().String()},
			TeamID:     team.ID,
			ContactID:  contact.ID,
			CampaignID: campaign.ID,
			Status:     models.EmailStatusDelivered,
			OpenedAt:   &now,
			Device:     "mobile",
		},
	}

	for _, tracking := range trackings {
		require.NoError(t, db.Create(&tracking).Error)
	}

	// Aggregate campaign analytics
	ctx := context.Background()
	analytics, err := aggregator.AggregateCampaignAnalytics(ctx, campaign.ID)
	require.NoError(t, err)

	// Verify results
	assert.Equal(t, int64(3), analytics.TotalSent)
	assert.Equal(t, 100.0, analytics.OpenRate) // All opened

	// Check device breakdown
	assert.Contains(t, analytics.DeviceBreakdown, "mobile")
	assert.Contains(t, analytics.DeviceBreakdown, "desktop")
	assert.Equal(t, int64(2), analytics.DeviceBreakdown["mobile"])
	assert.Equal(t, int64(1), analytics.DeviceBreakdown["desktop"])
}

func TestAnalyticsAggregator_AggregateContactInsights(t *testing.T) {
	db := setupTestDB(t)
	aggregator := NewAnalyticsAggregator(db)

	team := createTestTeam(t, db)
	contact := createTestContact(t, db, team.ID)
	campaign := createTestCampaign(t, db, team.ID)

	// Create email tracking for contact
	now := time.Now()
	trackings := []models.EmailTracking{
		{
			Base:       models.Base{ID: uuid.New().String()},
			TeamID:     team.ID,
			ContactID:  contact.ID,
			CampaignID: campaign.ID,
			Status:     models.EmailStatusDelivered,
			OpenedAt:   &now,
			ClickedAt:  &now,
		},
		{
			Base:       models.Base{ID: uuid.New().String()},
			TeamID:     team.ID,
			ContactID:  contact.ID,
			CampaignID: campaign.ID,
			Status:     models.EmailStatusDelivered,
			OpenedAt:   &now,
		},
		{
			Base:       models.Base{ID: uuid.New().String()},
			TeamID:     team.ID,
			ContactID:  contact.ID,
			CampaignID: campaign.ID,
			Status:     models.EmailStatusDelivered,
		},
	}

	for _, tracking := range trackings {
		require.NoError(t, db.Create(&tracking).Error)
	}

	// Aggregate contact insights
	ctx := context.Background()
	insights, err := aggregator.AggregateContactInsights(ctx, contact.ID)
	require.NoError(t, err)

	// Verify results
	assert.Equal(t, contact.ID, insights.ContactID)
	assert.Equal(t, int64(3), insights.TotalEmailsReceived)
	assert.Equal(t, int64(2), insights.TotalOpens)
	assert.Equal(t, int64(1), insights.TotalClicks)
	assert.InDelta(t, 66.67, insights.EngagementScore, 0.5) // (2+1)/3 * 100
}

func TestAnalyticsAggregator_CalculateEngagementScore(t *testing.T) {
	db := setupTestDB(t)
	aggregator := NewAnalyticsAggregator(db)

	tests := []struct {
		name           string
		totalReceived  int64
		totalOpens     int64
		totalClicks    int64
		expectedScore  float64
	}{
		{
			name:          "High engagement",
			totalReceived: 10,
			totalOpens:    8,
			totalClicks:   5,
			expectedScore: 65.0, // ((8 + 5) / 10) * 100
		},
		{
			name:          "Medium engagement",
			totalReceived: 10,
			totalOpens:    5,
			totalClicks:   2,
			expectedScore: 35.0,
		},
		{
			name:          "Low engagement",
			totalReceived: 10,
			totalOpens:    1,
			totalClicks:   0,
			expectedScore: 5.0,
		},
		{
			name:          "No emails",
			totalReceived: 0,
			totalOpens:    0,
			totalClicks:   0,
			expectedScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := aggregator.calculateEngagementScore(tt.totalReceived, tt.totalOpens, tt.totalClicks)
			assert.InDelta(t, tt.expectedScore, score, 0.1)
		})
	}
}

func TestKnowledgeBase_BuildTeamContext(t *testing.T) {
	db := setupTestDB(t)
	kb := NewKnowledgeBase(db)

	team := createTestTeam(t, db)
	contact := createTestContact(t, db, team.ID)
	campaign := createTestCampaign(t, db, team.ID)

	// Create tracking data
	now := time.Now()
	tracking := &models.EmailTracking{
		Base:       models.Base{ID: uuid.New().String()},
		TeamID:     team.ID,
		ContactID:  contact.ID,
		CampaignID: campaign.ID,
		Status:     models.EmailStatusDelivered,
		OpenedAt:   &now,
		ClickedAt:  &now,
	}
	require.NoError(t, db.Create(tracking).Error)

	// Build context
	ctx := context.Background()
	context, err := kb.BuildTeamContext(ctx, team.ID, 30)
	require.NoError(t, err)

	// Verify context contains key information
	assert.Contains(t, context, "Email Analytics")
	assert.Contains(t, context, "Total Sent:")
	assert.Contains(t, context, "Open Rate:")
	assert.Contains(t, context, "Click Rate:")
}

func TestKnowledgeBase_BuildCampaignContext(t *testing.T) {
	db := setupTestDB(t)
	kb := NewKnowledgeBase(db)

	team := createTestTeam(t, db)
	contact := createTestContact(t, db, team.ID)
	campaign := createTestCampaign(t, db, team.ID)

	// Create tracking data
	now := time.Now()
	tracking := &models.EmailTracking{
		Base:       models.Base{ID: uuid.New().String()},
		TeamID:     team.ID,
		ContactID:  contact.ID,
		CampaignID: campaign.ID,
		Status:     models.EmailStatusDelivered,
		OpenedAt:   &now,
		Device:     "mobile",
	}
	require.NoError(t, db.Create(tracking).Error)

	// Build context
	ctx := context.Background()
	context, err := kb.BuildCampaignContext(ctx, campaign.ID)
	require.NoError(t, err)

	// Verify context contains campaign information
	assert.Contains(t, context, "Campaign:")
	assert.Contains(t, context, campaign.Name)
	assert.Contains(t, context, "Subject:")
	assert.Contains(t, context, campaign.Subject)
	assert.Contains(t, context, "Performance")
}

func TestKnowledgeBase_BuildContactContext(t *testing.T) {
	db := setupTestDB(t)
	kb := NewKnowledgeBase(db)

	team := createTestTeam(t, db)
	contact := createTestContact(t, db, team.ID)
	campaign := createTestCampaign(t, db, team.ID)

	// Create tracking data
	now := time.Now()
	tracking := &models.EmailTracking{
		Base:       models.Base{ID: uuid.New().String()},
		TeamID:     team.ID,
		ContactID:  contact.ID,
		CampaignID: campaign.ID,
		Status:     models.EmailStatusDelivered,
		OpenedAt:   &now,
		ClickedAt:  &now,
	}
	require.NoError(t, db.Create(tracking).Error)

	// Build context
	ctx := context.Background()
	context, err := kb.BuildContactContext(ctx, contact.ID)
	require.NoError(t, err)

	// Verify context contains contact information
	assert.Contains(t, context, "Contact:")
	assert.Contains(t, context, contact.Email)
	assert.Contains(t, context, contact.Name)
	assert.Contains(t, context, "Engagement Score:")
}

func TestKnowledgeBase_ClassifyEngagement(t *testing.T) {
	db := setupTestDB(t)
	kb := NewKnowledgeBase(db)

	tests := []struct {
		score    float64
		expected string
	}{
		{80.0, "High"},
		{65.0, "High"},
		{60.0, "High"},
		{50.0, "Medium"},
		{40.0, "Medium"},
		{30.0, "Medium"},
		{20.0, "Low"},
		{10.0, "Low"},
		{0.0, "Low"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := kb.classifyEngagement(tt.score)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestKnowledgeBase_BenchmarkPerformance(t *testing.T) {
	db := setupTestDB(t)
	kb := NewKnowledgeBase(db)

	tests := []struct {
		openRate float64
		expected string
	}{
		{25.0, "Above average (industry average: ~21%)"},
		{30.0, "Above average (industry average: ~21%)"},
		{20.0, "Average (industry average: ~21%)"},
		{21.0, "Average (industry average: ~21%)"},
		{15.0, "Below average (industry average: ~21%)"},
		{10.0, "Below average (industry average: ~21%)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := kb.benchmarkPerformance(tt.openRate)
			assert.Equal(t, tt.expected, result)
		})
	}
}
