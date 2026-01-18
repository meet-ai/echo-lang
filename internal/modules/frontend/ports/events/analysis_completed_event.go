package events

import (
	"time"
)

// AnalysisCompletedEvent represents the event fired when analysis is completed
type AnalysisCompletedEvent struct {
	// Event metadata
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`

	// Analysis context
	SourceFileID string `json:"source_file_id"`
	AnalysisType string `json:"analysis_type"` // "lexical", "syntax", "semantic"

	// Analysis results
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`

	// Analysis artifacts
	TokenCount   int `json:"token_count,omitempty"`
	ASTNodeCount int `json:"ast_node_count,omitempty"`
	SymbolCount  int `json:"symbol_count,omitempty"`

	// Timing information
	Duration time.Duration `json:"duration"`
}

// NewAnalysisCompletedEvent creates a new analysis completed event
func NewAnalysisCompletedEvent(sourceFileID, analysisType string, success bool, errorMsg string, duration time.Duration) *AnalysisCompletedEvent {
	return &AnalysisCompletedEvent{
		EventID:   generateEventID(),
		EventType: "frontend.analysis.completed",
		Timestamp: time.Now(),
		SourceFileID: sourceFileID,
		AnalysisType: analysisType,
		Success:      success,
		Error:        errorMsg,
		Duration:     duration,
	}
}

// GetEventID returns the event ID
func (e *AnalysisCompletedEvent) GetEventID() string {
	return e.EventID
}

// GetEventType returns the event type
func (e *AnalysisCompletedEvent) GetEventType() string {
	return e.EventType
}

// OccurredAt returns when the event occurred
func (e *AnalysisCompletedEvent) OccurredAt() time.Time {
	return e.Timestamp
}

// AggregateID returns the aggregate ID this event relates to
func (e *AnalysisCompletedEvent) AggregateID() string {
	return e.SourceFileID
}

// generateEventID generates a unique event ID
func generateEventID() string {
	// Implementation would use UUID generation
	return "frontend-event-" + time.Now().Format("20060102-150405-000000")
}
