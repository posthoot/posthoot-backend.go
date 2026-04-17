package processors

import (
	"encoding/json"
	"fmt"
	"kori/internal/automation"
	"kori/internal/models"

	"gorm.io/gorm"
)

// AddToListProcessor handles ADD_TO_LIST nodes
type AddToListProcessor struct {
	db *gorm.DB
}

// AddToListNodeData represents the data structure for ADD_TO_LIST nodes
type AddToListNodeData struct {
	ListID string `json:"listId"` // Mailing list ID to add contact to
}

// NewAddToListProcessor creates a new ADD_TO_LIST node processor
func NewAddToListProcessor(db *gorm.DB) *AddToListProcessor {
	return &AddToListProcessor{db: db}
}

// Type returns the node type this processor handles
func (p *AddToListProcessor) Type() models.NodeType {
	return models.NodeTypeAddToList
}

// Validate checks if the ADD_TO_LIST node data is valid
func (p *AddToListProcessor) Validate(node *models.AutomationNode) error {
	if node.Type != models.NodeTypeAddToList {
		return fmt.Errorf("invalid node type for AddToListProcessor")
	}

	var data AddToListNodeData
	if err := json.Unmarshal(node.Data, &data); err != nil {
		return fmt.Errorf("invalid add_to_list node data: %w", err)
	}

	if data.ListID == "" {
		return fmt.Errorf("listId is required")
	}

	return nil
}

// Process executes the ADD_TO_LIST node logic
func (p *AddToListProcessor) Process(ctx *automation.ExecutionContext, node *models.AutomationNode) (*automation.ProcessResult, error) {
	var data AddToListNodeData
	if err := json.Unmarshal(node.Data, &data); err != nil {
		return nil, fmt.Errorf("failed to parse add_to_list node data: %w", err)
	}

	// Verify the list exists and belongs to the same team
	var list models.MailingList
	if err := p.db.Where("id = ? AND team_id = ?", data.ListID, ctx.TeamID).First(&list).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("mailing list not found or access denied")
		}
		return nil, fmt.Errorf("failed to load mailing list: %w", err)
	}

	// Check if contact is already in the list
	if ctx.Contact.ListID == data.ListID {
		// Contact already in this list, continue
		var edges []models.AutomationNodeEdge
		if err := p.db.Where("automation_id = ? AND source_id = ?", ctx.AutomationID, node.ID).Find(&edges).Error; err != nil {
			return nil, fmt.Errorf("failed to load edges: %w", err)
		}

		nextNodeIDs := make([]string, len(edges))
		for i, edge := range edges {
			nextNodeIDs[i] = edge.TargetID
		}

		return &automation.ProcessResult{
			NextNodeIDs: nextNodeIDs,
			Message:     fmt.Sprintf("Contact already in list '%s'", list.Name),
			Data: map[string]interface{}{
				"listId":   list.ID,
				"listName": list.Name,
				"skipped":  true,
			},
		}, nil
	}

	// Update contact's list
	if err := p.db.Model(ctx.Contact).Update("list_id", data.ListID).Error; err != nil {
		return nil, fmt.Errorf("failed to add contact to list: %w", err)
	}

	// Update list subscriber count
	if err := p.db.Model(&list).Update("subscribers_count", gorm.Expr("subscribers_count + ?", 1)).Error; err != nil {
		return nil, fmt.Errorf("failed to update list count: %w", err)
	}

	// Get next nodes
	var edges []models.AutomationNodeEdge
	if err := p.db.Where("automation_id = ? AND source_id = ?", ctx.AutomationID, node.ID).Find(&edges).Error; err != nil {
		return nil, fmt.Errorf("failed to load edges: %w", err)
	}

	nextNodeIDs := make([]string, len(edges))
	for i, edge := range edges {
		nextNodeIDs[i] = edge.TargetID
	}

	return &automation.ProcessResult{
		NextNodeIDs: nextNodeIDs,
		Message:     fmt.Sprintf("Added contact to list '%s'", list.Name),
		UpdateVars: map[string]interface{}{
			"current_list_id":   data.ListID,
			"current_list_name": list.Name,
		},
		Data: map[string]interface{}{
			"listId":   list.ID,
			"listName": list.Name,
		},
	}, nil
}
