package context

import (
	"bitbucket.org/free5gc-team/openapi/models"
)

// PCCRule - Policy and Charging Rule
type PCCRule struct {
	*models.PccRule

	// Reference Data
	refTrafficControlData string

	// related Data
	Datapath *DataPath
}

// NewPCCRule - create PCC rule from OpenAPI models
func NewPCCRule(model *models.PccRule) *PCCRule {
	if model == nil {
		return nil
	}

	r := &PCCRule{
		PccRule: model,
	}

	if model.RefTcData != nil {
		// TODO: now 1 pcc rule only maps to 1 TC data
		r.refTrafficControlData = model.RefTcData[0]
	}

	return r
}

// SetRefTrafficControlData - setting reference traffic control data
func (r *PCCRule) SetRefTrafficControlData(tcID string) {
	r.refTrafficControlData = tcID
}

// RefTrafficControlData - returns reference traffic control data ID
func (r *PCCRule) RefTrafficControlData() string {
	return r.refTrafficControlData
}
