package models

import "time"

// MessageStats holds aggregated message counts by status.
type MessageStats struct {
	TotalMessages int64 `json:"total_messages"`
	SuccessCount  int64 `json:"success_count"`
	FailedCount   int64 `json:"failed_count"`
}

// MessageStatsPerAgent holds aggregated message counts for a single chat agent.
type MessageStatsPerAgent struct {
	ChatAgentID   string `json:"chat_agent_id"`
	TotalMessages int64  `json:"total_messages"`
	SuccessCount  int64  `json:"success_count"`
	FailedCount   int64  `json:"failed_count"`
}

// MessageStatsResult holds the full grouped result of a message stats query.
type MessageStatsResult struct {
	Aggregate MessageStats           `json:"aggregate"`
	PerAgent  []MessageStatsPerAgent `json:"per_agent"`
}

// MessageStatsFilter defines filters for message stats queries.
type MessageStatsFilter struct {
	ChatAgentIDs []string
	From         time.Time
	To           time.Time
}
