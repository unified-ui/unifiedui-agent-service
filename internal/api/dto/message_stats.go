package dto

import (
	"strings"
	"time"
)

// MessageStatsRequest represents the query parameters for message stats.
type MessageStatsRequest struct {
	ChatAgentID string `form:"chat_agent_id"`
	From        string `form:"from"`
	To          string `form:"to"`
}

// ParseChatAgentIDs returns the comma-separated chat agent IDs as a slice.
// Empty entries are filtered out. Returns nil when no IDs are provided.
func (r *MessageStatsRequest) ParseChatAgentIDs() []string {
	if r.ChatAgentID == "" {
		return nil
	}
	parts := strings.Split(r.ChatAgentID, ",")
	ids := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			ids = append(ids, trimmed)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	return ids
}

// ParseTimeRange parses from/to strings into time.Time values.
func (r *MessageStatsRequest) ParseTimeRange() (from, to time.Time, err error) {
	if r.From != "" {
		from, err = time.Parse(time.RFC3339, r.From)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	if r.To != "" {
		to, err = time.Parse(time.RFC3339, r.To)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	return from, to, nil
}

// MessageStatsAggregate represents the overall aggregate counts.
type MessageStatsAggregate struct {
	TotalMessages int64 `json:"total_messages"`
	SuccessCount  int64 `json:"success_count"`
	FailedCount   int64 `json:"failed_count"`
}

// MessageStatsPerAgent represents counts for a single chat agent.
type MessageStatsPerAgent struct {
	ChatAgentID   string `json:"chat_agent_id"`
	TotalMessages int64  `json:"total_messages"`
	SuccessCount  int64  `json:"success_count"`
	FailedCount   int64  `json:"failed_count"`
}

// MessageStatsResponse represents the response for message stats.
type MessageStatsResponse struct {
	Aggregate MessageStatsAggregate  `json:"aggregate"`
	PerAgent  []MessageStatsPerAgent `json:"per_agent"`
}
