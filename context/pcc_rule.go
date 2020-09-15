package context

import (
	"bitbucket.org/free5gc-team/openapi/models"
)

// PCCRule - Policy and Charging Rule
type PCCRule struct {
	// shall include attribute
	PCCRuleID  string
	Precedence int32

	// maybe include attribute
	AppID     string
	FlowInfos []models.FlowInformation

	// Reference Data
	refTrafficControlData string
}

// NewPCCRuleFromModel - create PCC rule from OpenAPI models
func NewPCCRuleFromModel(pccModel *models.PccRule) *PCCRule {
	if pccModel == nil {
		return nil
	}
	pccRule := new(PCCRule)

	pccRule.PCCRuleID = pccModel.PccRuleId
	pccRule.Precedence = pccModel.Precedence
	pccRule.AppID = pccModel.AppId
	pccRule.FlowInfos = pccModel.FlowInfos

	return pccRule
}

// SetRefTrafficControlData - setting reference traffic control data
func (r *PCCRule) SetRefTrafficControlData(tcID string) {
	r.refTrafficControlData = tcID
}

// RefTrafficControlData - returns refernece traffic control data ID
func (r *PCCRule) RefTrafficControlData() string {
	return r.refTrafficControlData
}
