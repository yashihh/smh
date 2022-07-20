package context

import (
	"bitbucket.org/free5gc-team/openapi/models"
)

// SessionRule - A session rule consists of policy information elements
// associated with PDU session.
type SessionRule struct {
	*models.SessionRule
	DefQosQFI uint8
}

// NewSessionRule - create session rule from OpenAPI models
func NewSessionRule(model *models.SessionRule) *SessionRule {
	if model == nil {
		return nil
	}

	return &SessionRule{
		SessionRule: model,
		// now use 5QI as QFI
		// TODO: dynamic QFI
		DefQosQFI: uint8(model.AuthDefQos.Var5qi),
	}
}
