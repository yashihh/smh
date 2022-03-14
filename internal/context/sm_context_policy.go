package context

import (
	"fmt"
	"reflect"

	"bitbucket.org/free5gc-team/openapi/models"
	"bitbucket.org/free5gc-team/smf/pkg/factory"
)

// SM Policy related operation

// SelectedSessionRule - return the SMF selected session rule for this SM Context
func (c *SMContext) SelectedSessionRule() *SessionRule {
	if c.SelectedSessionRuleID == "" {
		return nil
	}
	return c.SessionRules[c.SelectedSessionRuleID]
}

func (c *SMContext) ApplySessionRules(
	decision *models.SmPolicyDecision) error {
	if decision == nil {
		return fmt.Errorf("SmPolicyDecision is nil")
	}

	for id, r := range decision.SessRules {
		if r == nil {
			c.Log.Debugf("Delete SessionRule[%s]", id)
			delete(c.SessionRules, id)
		} else {
			if origRule, ok := c.SessionRules[id]; ok {
				c.Log.Debugf("Modify SessionRule[%s]: %+v", id, r)
				origRule.SessionRule = r
			} else {
				c.Log.Debugf("Install SessionRule[%s]: %+v", id, r)
				c.SessionRules[id] = NewSessionRule(r)
			}
		}
	}

	if len(c.SessionRules) == 0 {
		return fmt.Errorf("ApplySessionRules: no session rule")
	}

	if c.SelectedSessionRuleID == "" ||
		c.SessionRules[c.SelectedSessionRuleID] == nil {
		// No active session rule, randomly choose a session rule to activate
		for id := range c.SessionRules {
			c.SelectedSessionRuleID = id
			break
		}
	}

	// TODO: need update default data path if SessionRule changed
	return nil
}

func (c *SMContext) ApplyPccRules(
	decision *models.SmPolicyDecision) error {
	if decision == nil {
		return fmt.Errorf("SmPolicyDecision is nil")
	}

	finalPccRules := make(map[string]*PCCRule)
	finalTcDatas := make(map[string]*TrafficControlData)
	finalQosDatas := make(map[string]*models.QosData)

	// Handle PccRules in decision first
	for id, pccModel := range decision.PccRules {
		var srcTcData, tgtTcData *TrafficControlData
		srcPcc, ok := c.PCCRules[id]
		if pccModel == nil {
			c.Log.Infof("Remove PCCRule[%s]", id)
			if !ok {
				c.Log.Warnf("PCCRule[%s] not exist", id)
				continue
			}

			srcTcData = c.TrafficControlDatas[srcPcc.RefTcDataID()]
			c.PreRemoveDataPath(srcPcc.Datapath)
		} else {
			tgtPcc := NewPCCRule(pccModel)
			tgtTcID := tgtPcc.RefTcDataID()
			tgtQosID := tgtPcc.RefQosDataID()
			tgtTcData = NewTrafficControlData(
				decision.TraffContDecs[tgtTcID])
			tgtQosData := decision.QosDecs[tgtQosID]

			// Create Data path for targetPccRule
			if err := c.CreatePccRuleDataPath(tgtPcc, tgtTcData, tgtQosData); err != nil {
				return err
			}
			if ok {
				c.Log.Infof("Modify PCCRule[%s]", id)
				srcTcData = c.TrafficControlDatas[srcPcc.RefTcDataID()]
				c.PreRemoveDataPath(srcPcc.Datapath)
			} else {
				c.Log.Infof("Install PCCRule[%s]", id)
			}

			if err := applyFlowInfoOrPFD(tgtPcc); err != nil {
				return err
			}
			finalPccRules[id] = tgtPcc
			finalTcDatas[tgtTcID] = tgtTcData
			finalQosDatas[tgtQosID] = tgtQosData
		}
		if err := checkUpPathChgEvent(c, srcTcData, tgtTcData); err != nil {
			c.Log.Warnf("Check UpPathChgEvent err: %v", err)
		}
		// Remove source pcc rule
		delete(c.PCCRules, id)
	}

	// Handle PccRules not in decision
	for id, pcc := range c.PCCRules {
		tcDataID := pcc.RefTcDataID()
		qosDataID := pcc.RefQosDataID()
		srcTcData := c.TrafficControlDatas[tcDataID]
		tgtTcData := NewTrafficControlData(
			decision.TraffContDecs[tcDataID])
		srcQosData := c.QosDatas[qosDataID]
		tgtQosData := decision.QosDecs[qosDataID]
		if !reflect.DeepEqual(srcTcData, tgtTcData) ||
			!reflect.DeepEqual(srcQosData, tgtQosData) {
			// Remove old Data path
			c.PreRemoveDataPath(pcc.Datapath)
			// Create new Data path
			if err := c.CreatePccRuleDataPath(pcc, tgtTcData, tgtQosData); err != nil {
				return err
			}
			if err := checkUpPathChgEvent(c, srcTcData, tgtTcData); err != nil {
				c.Log.Warnf("Check UpPathChgEvent err: %v", err)
			}
		}
		finalPccRules[id] = pcc
		finalTcDatas[tcDataID] = tgtTcData
		finalQosDatas[qosDataID] = tgtQosData
	}

	c.PCCRules = finalPccRules
	c.TrafficControlDatas = finalTcDatas
	c.QosDatas = finalQosDatas
	return nil
}

// Set data path PDR to REMOVE beofre sending PFCP req
func (c *SMContext) PreRemoveDataPath(dp *DataPath) {
	if dp == nil {
		return
	}
	dp.RemovePDR()
	c.DataPathToBeRemoved[dp.PathID] = dp
}

// Remove data path after receiving PFCP rsp
func (c *SMContext) PostRemoveDataPath() {
	for id, dp := range c.DataPathToBeRemoved {
		dp.DeactivateTunnelAndPDR(c)
		c.Tunnel.RemoveDataPath(id)
		delete(c.DataPathToBeRemoved, id)
	}
}

func applyFlowInfoOrPFD(pcc *PCCRule) error {
	appID := pcc.AppId

	if len(pcc.FlowInfos) == 0 && appID == "" {
		return fmt.Errorf("No FlowInfo and AppID")
	}

	// Apply flow description if it presents
	if flowDesc := pcc.FlowDescription(); flowDesc != "" {
		if err := pcc.UpdateDataPathFlowDescription(flowDesc); err != nil {
			return err
		}
		return nil
	}

	// Find PFD with AppID if no flow description presents
	// TODO: Get PFD from NEF (not from config)
	var matchedPFD *factory.PfdDataForApp
	for _, pfdDataForApp := range factory.UERoutingConfig.PfdDatas {
		if pfdDataForApp.AppID == appID {
			matchedPFD = pfdDataForApp
			break
		}
	}

	// Workaround: UPF doesn't support PFD. Get FlowDescription from PFD.
	// TODO: Send PFD to UPF
	if matchedPFD == nil ||
		len(matchedPFD.Pfds) == 0 ||
		len(matchedPFD.Pfds[0].FlowDescriptions) == 0 {
		return fmt.Errorf("No PFD matched for AppID [%s]", appID)
	}
	if err := pcc.UpdateDataPathFlowDescription(
		matchedPFD.Pfds[0].FlowDescriptions[0]); err != nil {
		return err
	}
	return nil
}

func checkUpPathChgEvent(c *SMContext,
	srcTcData, tgtTcData *TrafficControlData) error {
	var srcRoute, tgtRoute models.RouteToLocation
	var upPathChgEvt *models.UpPathChgEvent

	if srcTcData == nil && tgtTcData == nil {
		c.Log.Infof("No srcTcData and tgtTcData. Nothing to do")
		return nil
	}

	// Set reference to traffic control data
	if srcTcData != nil {
		if len(srcTcData.RouteToLocs) == 0 {
			return fmt.Errorf("No RouteToLocs in srcTcData")
		}
		// TODO: Fix always choosing the first RouteToLocs as source Route
		srcRoute = srcTcData.RouteToLocs[0]
		// If no target TcData, the default UpPathChgEvent will be the one in source TcData
		upPathChgEvt = srcTcData.UpPathChgEvent
	} else {
		// No source TcData, get default route
		srcRoute = models.RouteToLocation{
			Dnai: "",
		}
	}

	if tgtTcData != nil {
		if len(tgtTcData.RouteToLocs) == 0 {
			return fmt.Errorf("No RouteToLocs in tgtTcData")
		}
		// TODO: Fix always choosing the first RouteToLocs as target Route
		tgtRoute = tgtTcData.RouteToLocs[0]
		// If target TcData is available, change UpPathChgEvent to the one in target TcData
		upPathChgEvt = tgtTcData.UpPathChgEvent
	} else {
		// No target TcData in decision, roll back to the default route
		tgtRoute = models.RouteToLocation{
			Dnai: "",
		}
	}

	if !reflect.DeepEqual(srcRoute, tgtRoute) {
		c.BuildUpPathChgEventExposureNotification(upPathChgEvt, &srcRoute, &tgtRoute)
	}

	return nil
}
