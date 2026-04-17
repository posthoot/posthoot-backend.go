package automation

import (
	"kori/internal/models"
	"time"
)

// ExecutionContext holds the state during automation execution
type ExecutionContext struct {
	AutomationID string
	ContactID    string
	TeamID       string
	Variables    map[string]interface{}
	CurrentNode  *models.AutomationNode
	ExecutionLog []LogEntry
	Contact      *models.Contact
	Execution    *models.AutomationExecution
}

// LogEntry represents a single node execution in the log
type LogEntry struct {
	NodeID      string                 `json:"nodeId"`
	NodeType    models.NodeType        `json:"nodeType"`
	Timestamp   time.Time              `json:"timestamp"`
	Status      string                 `json:"status"` // success, failed, skipped
	Message     string                 `json:"message"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Error       string                 `json:"error,omitempty"`
	DurationMs  int64                  `json:"durationMs"`
}

// AddLog adds an entry to the execution log
func (ctx *ExecutionContext) AddLog(nodeID string, nodeType models.NodeType, status string, message string, err error) {
	entry := LogEntry{
		NodeID:    nodeID,
		NodeType:  nodeType,
		Timestamp: time.Now(),
		Status:    status,
		Message:   message,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	ctx.ExecutionLog = append(ctx.ExecutionLog, entry)
}

// UpdateVariable updates or sets a variable in the execution context
func (ctx *ExecutionContext) UpdateVariable(key string, value interface{}) {
	if ctx.Variables == nil {
		ctx.Variables = make(map[string]interface{})
	}
	ctx.Variables[key] = value
}

// GetVariable retrieves a variable from the execution context
func (ctx *ExecutionContext) GetVariable(key string) (interface{}, bool) {
	if ctx.Variables == nil {
		return nil, false
	}
	val, exists := ctx.Variables[key]
	return val, exists
}

// NewExecutionContext creates a new execution context
func NewExecutionContext(automationID, contactID, teamID string, triggerData map[string]interface{}) *ExecutionContext {
	return &ExecutionContext{
		AutomationID: automationID,
		ContactID:    contactID,
		TeamID:       teamID,
		Variables:    triggerData,
		ExecutionLog: []LogEntry{},
	}
}
