package processors

import (
	"encoding/json"
	"fmt"
	"kori/internal/automation"
	"kori/internal/models"
	"time"

	"gorm.io/gorm"
)

// WaitProcessor handles WAIT nodes
type WaitProcessor struct {
	db *gorm.DB
}

// WaitNodeData represents the data structure for WAIT nodes
type WaitNodeData struct {
	Duration string `json:"duration"` // e.g., "5m", "1h", "30s"
}

// NewWaitProcessor creates a new WAIT node processor
func NewWaitProcessor(db *gorm.DB) *WaitProcessor {
	return &WaitProcessor{db: db}
}

// Type returns the node type this processor handles
func (p *WaitProcessor) Type() models.NodeType {
	return models.NodeTypeWait
}

// Validate checks if the WAIT node data is valid
func (p *WaitProcessor) Validate(node *models.AutomationNode) error {
	if node.Type != models.NodeTypeWait {
		return fmt.Errorf("invalid node type for WaitProcessor")
	}

	var data WaitNodeData
	if err := json.Unmarshal(node.Data, &data); err != nil {
		return fmt.Errorf("invalid wait node data: %w", err)
	}

	// Validate duration format
	if data.Duration == "" {
		return fmt.Errorf("duration is required")
	}

	if _, err := time.ParseDuration(data.Duration); err != nil {
		return fmt.Errorf("invalid duration format: %w", err)
	}

	return nil
}

// Process executes the WAIT node logic
func (p *WaitProcessor) Process(ctx *automation.ExecutionContext, node *models.AutomationNode) (*automation.ProcessResult, error) {
	var data WaitNodeData
	if err := json.Unmarshal(node.Data, &data); err != nil {
		return nil, fmt.Errorf("failed to parse wait node data: %w", err)
	}

	// Parse duration
	duration, err := time.ParseDuration(data.Duration)
	if err != nil {
		return nil, fmt.Errorf("failed to parse duration: %w", err)
	}

	// Get next nodes
	var edges []models.AutomationNodeEdge
	if err := p.db.Where("automation_id = ? AND source_id = ?", ctx.AutomationID, node.ID).Find(&edges).Error; err != nil {
		return nil, fmt.Errorf("failed to load edges: %w", err)
	}

	if len(edges) == 0 {
		return &automation.ProcessResult{
			Complete: true,
			Message:  fmt.Sprintf("WAIT node has no outgoing edges, workflow paused for %s", data.Duration),
		}, nil
	}

	nextNodeIDs := make([]string, len(edges))
	for i, edge := range edges {
		nextNodeIDs[i] = edge.TargetID
	}

	// Return result with wait duration
	return &automation.ProcessResult{
		NextNodeIDs: nextNodeIDs,
		Wait:        &duration,
		Message:     fmt.Sprintf("Waiting for %s before continuing", data.Duration),
		Data: map[string]interface{}{
			"duration":   data.Duration,
			"resumeAt":   time.Now().Add(duration).Format(time.RFC3339),
		},
	}, nil
}
