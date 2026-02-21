package traceimport

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/unifiedui/agent-service/internal/services/platform"
)

func TestAgentTypeFromJobType_Foundry(t *testing.T) {
	result := agentTypeFromJobType(JobTypeMicrosoftFoundry)
	assert.Equal(t, platform.AgentTypeFoundry, result)
}

func TestAgentTypeFromJobType_N8N(t *testing.T) {
	result := agentTypeFromJobType(JobTypeN8N)
	assert.Equal(t, platform.AgentTypeN8N, result)
}

func TestAgentTypeFromJobType_Unknown(t *testing.T) {
	result := agentTypeFromJobType(JobType("unknown"))
	assert.Equal(t, platform.AgentType("unknown"), result)
}

func TestJobTypeFromAgentType_Foundry(t *testing.T) {
	result := jobTypeFromAgentType(platform.AgentTypeFoundry)
	assert.Equal(t, JobTypeMicrosoftFoundry, result)
}

func TestJobTypeFromAgentType_N8N(t *testing.T) {
	result := jobTypeFromAgentType(platform.AgentTypeN8N)
	assert.Equal(t, JobTypeN8N, result)
}

func TestJobTypeFromAgentType_Unknown(t *testing.T) {
	result := jobTypeFromAgentType(platform.AgentType("custom"))
	assert.Equal(t, JobType("custom"), result)
}
