package processors

import (
	"encoding/json"
	"fmt"
	"kori/internal/automation"
	"kori/internal/models"

	"gorm.io/gorm"
)

// UpdateSubscriberProcessor handles UPDATE_SUBSCRIBER nodes
type UpdateSubscriberProcessor struct {
	db *gorm.DB
}

// UpdateSubscriberNodeData represents the data structure for UPDATE_SUBSCRIBER nodes
type UpdateSubscriberNodeData struct {
	Fields map[string]interface{} `json:"fields"` // Fields to update on the contact
}

// NewUpdateSubscriberProcessor creates a new UPDATE_SUBSCRIBER node processor
func NewUpdateSubscriberProcessor(db *gorm.DB) *UpdateSubscriberProcessor {
	return &UpdateSubscriberProcessor{db: db}
}

// Type returns the node type this processor handles
func (p *UpdateSubscriberProcessor) Type() models.NodeType {
	return models.NodeTypeUpdateSubscriber
}

// Validate checks if the UPDATE_SUBSCRIBER node data is valid
func (p *UpdateSubscriberProcessor) Validate(node *models.AutomationNode) error {
	if node.Type != models.NodeTypeUpdateSubscriber {
		return fmt.Errorf("invalid node type for UpdateSubscriberProcessor")
	}

	var data UpdateSubscriberNodeData
	if err := json.Unmarshal(node.Data, &data); err != nil {
		return fmt.Errorf("invalid update_subscriber node data: %w", err)
	}

	if len(data.Fields) == 0 {
		return fmt.Errorf("at least one field update is required")
	}

	// Validate allowed fields
	allowedFields := map[string]bool{
		"first_name": true,
		"last_name":  true,
		"company":    true,
		"country":    true,
		"city":       true,
		"state":      true,
		"zip":        true,
		"address":    true,
		"phone":      true,
		"linkedin":   true,
		"twitter":    true,
		"facebook":   true,
		"instagram":  true,
		"status":     true,
	}

	for field := range data.Fields {
		if !allowedFields[field] {
			return fmt.Errorf("field '%s' is not allowed for update", field)
		}
	}

	return nil
}

// Process executes the UPDATE_SUBSCRIBER node logic
func (p *UpdateSubscriberProcessor) Process(ctx *automation.ExecutionContext, node *models.AutomationNode) (*automation.ProcessResult, error) {
	var data UpdateSubscriberNodeData
	if err := json.Unmarshal(node.Data, &data); err != nil {
		return nil, fmt.Errorf("failed to parse update_subscriber node data: %w", err)
	}

	// Prepare updates map
	updates := make(map[string]interface{})
	for field, value := range data.Fields {
		// Handle variable substitution
		if strValue, ok := value.(string); ok {
			// Check if value references a variable
			if varValue, exists := ctx.GetVariable(strValue); exists {
				updates[field] = varValue
			} else {
				updates[field] = strValue
			}
		} else {
			updates[field] = value
		}
	}

	// Apply updates
	if err := p.db.Model(ctx.Contact).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update contact: %w", err)
	}

	// Reload contact to get updated values
	if err := p.db.Where("id = ?", ctx.ContactID).First(ctx.Contact).Error; err != nil {
		return nil, fmt.Errorf("failed to reload contact: %w", err)
	}

	// Update context variables with new contact data
	for field := range updates {
		switch field {
		case "first_name":
			ctx.UpdateVariable("contact_first_name", ctx.Contact.FirstName)
		case "last_name":
			ctx.UpdateVariable("contact_last_name", ctx.Contact.LastName)
		case "email":
			ctx.UpdateVariable("contact_email", ctx.Contact.Email)
		case "company":
			ctx.UpdateVariable("contact_company", ctx.Contact.Company)
		case "phone":
			ctx.UpdateVariable("contact_phone", ctx.Contact.Phone)
		}
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
		Message:     fmt.Sprintf("Updated %d field(s) on contact", len(updates)),
		Data: map[string]interface{}{
			"updatedFields": data.Fields,
			"contactId":     ctx.ContactID,
		},
	}, nil
}
