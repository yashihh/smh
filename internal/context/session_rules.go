package context

import (
	"bitbucket.org/free5gc-team/openapi/models"
)

// SessionRule - A session rule consists of policy information elements
// associated with PDU session.
type SessionRule struct {
	*models.SessionRule
}

// NewSessionRule - create session rule from OpenAPI models
func NewSessionRule(model *models.SessionRule) *SessionRule {
	if model == nil {
		return nil
	}

	return &SessionRule{
		SessionRule: model,
	}
}
