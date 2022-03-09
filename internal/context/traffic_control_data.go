package context

import (
	"bitbucket.org/free5gc-team/openapi/models"
)

// TrafficControlData - Traffic control data defines how traffic data flows
// associated with a rule are treated (e.g. blocked, redirected).
type TrafficControlData struct {
	*models.TrafficControlData

	// referenced dataType
	refedPCCRule map[string]string
}

// NewTrafficControlData - create the traffic control data from OpenAPI model
func NewTrafficControlData(model *models.TrafficControlData) *TrafficControlData {
	if model == nil {
		return nil
	}

	return &TrafficControlData{
		TrafficControlData: model,
		refedPCCRule:       make(map[string]string),
	}
}

// RefedPCCRules - returns the PCCRules that reference this tcData
func (tc *TrafficControlData) RefedPCCRules() map[string]string {
	return tc.refedPCCRule
}

func (tc *TrafficControlData) AddRefedPCCRules(PCCref string) {
	tc.refedPCCRule[PCCref] = PCCref
}

func (tc *TrafficControlData) DeleteRefedPCCRules(PCCref string) {
	delete(tc.refedPCCRule, PCCref)
}
