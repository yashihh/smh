package producer

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"bitbucket.org/free5gc-team/flowdesc"
	"bitbucket.org/free5gc-team/http_wrapper"
	"bitbucket.org/free5gc-team/openapi/Nsmf_EventExposure"
	"bitbucket.org/free5gc-team/openapi/models"
	"bitbucket.org/free5gc-team/pfcp/pfcpType"
	smf_context "bitbucket.org/free5gc-team/smf/context"
	"bitbucket.org/free5gc-team/smf/factory"
	"bitbucket.org/free5gc-team/smf/logger"
)

func HandleSMPolicyUpdateNotify(smContextRef string, request models.SmPolicyNotification) *http_wrapper.Response {
	logger.PduSessLog.Infoln("In HandleSMPolicyUpdateNotify")
	decision := request.SmPolicyDecision
	smContext := smf_context.GetSMContext(smContextRef)

	if smContext == nil {
		logger.PduSessLog.Errorf("SMContext[%s] not found", smContextRef)
		httpResponse := http_wrapper.NewResponse(http.StatusBadRequest, nil, nil)
		return httpResponse
	}

	smContext.SMLock.Lock()
	defer smContext.SMLock.Unlock()

	if smContext.SMContextState != smf_context.Active {
		//Wait till the state becomes Active again
		//TODO: implement waiting in concurrent architecture
		logger.PduSessLog.Warnf("SMContext[%s-%02d] should be Active, but actual %s",
			smContext.Supi, smContext.PDUSessionID, smContext.SMContextState.String())
	}

	//TODO: Response data type -
	//[200 OK] UeCampingRep
	//[200 OK] array(PartialSuccessReport)
	//[400 Bad Request] ErrorReport
	httpResponse := http_wrapper.NewResponse(http.StatusNoContent, nil, nil)
	if err := ApplySmPolicyFromDecision(smContext, decision); err != nil {
		logger.PduSessLog.Errorf("apply sm policy decision error: %+v", err)
		//TODO: Fill the error body
		httpResponse.Status = http.StatusBadRequest
	}

	return httpResponse
}

func SendUpPathChgEventExposureNotification(
	chgEvent *models.UpPathChgEvent, chgType string,
	sourceTR, targetTR *models.RouteToLocation) {
	if chgEvent == nil {
		logger.PduSessLog.Infof("UpPathChgEvent is nil")
		return
	}
	if chgEvent.NotificationUri == "" || !strings.Contains(string(chgEvent.DnaiChgType), chgType) {
		logger.PduSessLog.Infof("No NotificationUri [%s] or DnaiChgType [%s] not includes %s",
			chgEvent.NotificationUri, chgEvent.DnaiChgType, chgType)
		return
	}
	notification := models.NsmfEventExposureNotification{
		NotifId: chgEvent.NotifCorreId,
		EventNotifs: []models.EventNotification{
			{
				Event:            models.SmfEvent_UP_PATH_CH,
				DnaiChgType:      models.DnaiChangeType(chgType),
				SourceTraRouting: sourceTR,
				TargetTraRouting: targetTR,
			},
		},
	}
	if sourceTR.Dnai != targetTR.Dnai {
		notification.EventNotifs[0].SourceDnai = sourceTR.Dnai
		notification.EventNotifs[0].TargetDnai = targetTR.Dnai
	}
	//TODO: sourceUeIpv4Addr, sourceUeIpv6Prefix, targetUeIpv4Addr, targetUeIpv6Prefix

	logger.PduSessLog.Infof("Send UpPathChg Event Exposure Notification [%s] to NEF/AF", chgType)
	configuration := Nsmf_EventExposure.NewConfiguration()
	client := Nsmf_EventExposure.NewAPIClient(configuration)
	_, httpResponse, err := client.
		DefaultCallbackApi.
		SmfEventExposureNotification(context.Background(), chgEvent.NotificationUri, notification)
	if err != nil {
		if httpResponse != nil {
			logger.PduSessLog.Warnf("SMF Event Exposure Notification Error[%s]", httpResponse.Status)
		} else {
			logger.PduSessLog.Warnf("SMF Event Exposure Notification Failed[%s]", err.Error())
		}
		return
	} else if httpResponse == nil {
		logger.PduSessLog.Warnln("SMF Event Exposure Notification Failed[HTTP Response is nil]")
		return
	}
	if httpResponse.StatusCode != http.StatusOK && httpResponse.StatusCode != http.StatusNoContent {
		logger.PduSessLog.Warnf("SMF Event Exposure Notification Failed")
	} else {
		logger.PduSessLog.Tracef("SMF Event Exposure Notification Success")
	}
}

func handleSessionRule(smContext *smf_context.SMContext, id string, sessionRuleModel *models.SessionRule) {
	if sessionRuleModel == nil {
		logger.PduSessLog.Debugf("Delete SessionRule[%s]", id)
		delete(smContext.SessionRules, id)
	} else {
		sessRule := smf_context.NewSessionRuleFromModel(sessionRuleModel)
		// Session rule installation
		if oldSessRule, exist := smContext.SessionRules[id]; !exist {
			logger.PduSessLog.Debugf("Install SessionRule[%s]", id)
			smContext.SessionRules[id] = sessRule
		} else { // Session rule modification
			logger.PduSessLog.Debugf("Modify SessionRule[%s]", oldSessRule.SessionRuleID)
			smContext.SessionRules[id] = sessRule
		}
	}
}

func getTcDataFromDecision(pccRule *smf_context.PCCRule,
	decision *models.SmPolicyDecision) *smf_context.TrafficControlData {
	var tcData *smf_context.TrafficControlData
	if tcID := pccRule.RefTrafficControlData(); tcID != "" {
		if tcModel, exist := decision.TraffContDecs[tcID]; exist && tcModel != nil {
			tcData = smf_context.NewTrafficControlDataFromModel(tcModel)
			tcData.AddRefedPCCRules(pccRule.PCCRuleID)
		}
	}
	return tcData
}

func ApplySmPolicyFromDecision(smContext *smf_context.SMContext, decision *models.SmPolicyDecision) error {
	logger.PduSessLog.Traceln("In ApplySmPolicyFromDecision")
	var err error
	smContext.SMContextState = smf_context.ModificationPending
	selectedSessionRule := smContext.SelectedSessionRule()
	if selectedSessionRule == nil { //No active session rule
		//Update session rules from decision
		for id, sessRuleModel := range decision.SessRules {
			handleSessionRule(smContext, id, sessRuleModel)
		}
		for id := range smContext.SessionRules {
			// Randomly choose a session rule to activate
			smf_context.SetSessionRuleActivateState(smContext.SessionRules[id], true)
			break
		}
	} else {
		selectedSessionRuleID := selectedSessionRule.SessionRuleID
		//Update session rules from decision
		for id, sessRuleModel := range decision.SessRules {
			handleSessionRule(smContext, id, sessRuleModel)
		}
		if _, exist := smContext.SessionRules[selectedSessionRuleID]; !exist {
			//Original active session rule is deleted; choose again
			for id := range smContext.SessionRules {
				// Randomly choose a session rule to activate
				smf_context.SetSessionRuleActivateState(smContext.SessionRules[id], true)
				break
			}
		} else {
			//Activate original active session rule
			smf_context.SetSessionRuleActivateState(smContext.SessionRules[selectedSessionRuleID], true)
		}
	}

	if len(decision.PccRules) == 0 {
		// No pcc rule, only traffic control Data
		for tcID, tcModel := range decision.TraffContDecs {
			if tcModel == nil {
				logger.PduSessLog.Infof("Remove Traffic Control Data[%s] in SMContext[%s]", tcID, smContext.Ref)
			} else {
				logger.PduSessLog.Infof("Modify Traffic Control Data[%s] in SMContext[%s]", tcID, smContext.Ref)
			}
			if tcData, exist := smContext.TrafficControlPool[tcID]; exist {
				for _, pccRef := range tcData.RefedPCCRules() {
					relatedPCCRule := smContext.PCCRules[pccRef]
					if err = applyTrafficRoutingData(smContext, relatedPCCRule, relatedPCCRule,
						relatedPCCRule.RefTrafficControlData(),
						getTcDataFromDecision(relatedPCCRule, decision)); err != nil {
						return err
					}
				}
			} else {
				logger.PduSessLog.Errorf("Traffic Control Data[%s] not exist in SMContext[%s]", tcID, smContext.Ref)
			}
		}
		return err
	}

	for pccRuleID, pccRuleModel := range decision.PccRules {
		srcPccRule, exist := smContext.PCCRules[pccRuleID]
		var qos *models.QosData
		if pccRuleModel.RefQosData != nil {
			qos = decision.QosDecs[pccRuleModel.RefQosData[0]]
		}
		if pccRuleModel == nil {
			logger.PduSessLog.Infof("Remove PCCRule[%s] in SMContext[%s]", pccRuleID, smContext.Ref)
			if !exist {
				logger.PduSessLog.Errorf("PCCRule[%s] not exist in SMContext[%s]", pccRuleID, smContext.Ref)
				continue
			}
			err = applyPCCRule(smContext, srcPccRule, nil, pccRuleID, nil, qos)
		} else {
			if exist {
				logger.PduSessLog.Infof("Modify PCCRule[%s] in SMContext[%s]", pccRuleID, smContext.Ref)
			} else {
				logger.PduSessLog.Infof("Install PCCRule[%s] in SMContext[%s]", pccRuleID, smContext.Ref)
			}
			targetPccRule := smf_context.NewPCCRuleFromModel(pccRuleModel)
			err = applyPCCRule(
				smContext, srcPccRule, targetPccRule, pccRuleID,
				getTcDataFromDecision(targetPccRule, decision), qos)
		}
	}

	logger.PduSessLog.Traceln("End of ApplySmPolicyFromDecision")
	return err
}

func applyPCCRule(smContext *smf_context.SMContext, srcPccRule, targetPccRule *smf_context.PCCRule,
	pccRuleID string, tcData *smf_context.TrafficControlData, qos *models.QosData) error {
	if targetPccRule == nil { //Remove PCC Rule
		if srcPccRule != nil {
			if err := applyTrafficRoutingData(smContext, srcPccRule, nil, srcPccRule.RefTrafficControlData(), nil); err != nil {
				return err
			}
			delete(smContext.PCCRules, pccRuleID)
		} else {
			logger.PduSessLog.Errorf("srcPccRule[%s] not exist in SMContext[%s]", pccRuleID, smContext.Ref)
		}
		return nil
	}

	//Create Data path for targetPccRule
	createPccRuleDataPath(smContext, targetPccRule, tcData)
	addQoSToDataPath(smContext, targetPccRule.Datapath, qos)

	if appID := targetPccRule.AppID; appID != "" {
		var matchedPFD *factory.PfdDataForApp
		for _, pfdDataForApp := range factory.UERoutingConfig.PfdDatas {
			if pfdDataForApp.AppID == appID {
				matchedPFD = pfdDataForApp
				break
			}
		}

		//TODO: support multiple PFDs and FlowDescriptions
		if matchedPFD != nil && matchedPFD.Pfds != nil && matchedPFD.Pfds[0].FlowDescriptions != nil {
			flowDescConfig := matchedPFD.Pfds[0].FlowDescriptions[0]
			uplinkFlowDescription := getUplinkFlowDescription(flowDescConfig)

			if uplinkFlowDescription == "" {
				return fmt.Errorf("getUplinkFlowDescription failed")
			} else {
				updatePccRuleDataPathFlowDescription(targetPccRule, flowDescConfig, uplinkFlowDescription)
			}
		} else {
			logger.PduSessLog.Errorf("No PFD matched for Aplicationp ID [%s]", appID)
		}
	}

	if targetPccRule.FlowInfos != nil {
		flowDescConfig := targetPccRule.FlowInfos[0].FlowDescription
		uplinkFlowDescription := getUplinkFlowDescription(flowDescConfig)
		if uplinkFlowDescription == "" {
			return fmt.Errorf("getUplinkFlowDescription failed")
		} else {
			updatePccRuleDataPathFlowDescription(targetPccRule, flowDescConfig, uplinkFlowDescription)
		}
	}

	if err := applyTrafficRoutingData(smContext, srcPccRule, targetPccRule,
		targetPccRule.RefTrafficControlData(), tcData); err != nil {
		return err
	}

	// Install PCC rule
	smContext.PCCRules[pccRuleID] = targetPccRule

	return nil
}

func applyTrafficRoutingData(smContext *smf_context.SMContext, srcPccRule, targetPccRule *smf_context.PCCRule,
	applyTcID string, targetTcData *smf_context.TrafficControlData) error {
	trChanged := false
	var srcTraRouting, targetTraRouting models.RouteToLocation
	var upPathChgEvt *models.UpPathChgEvent

	if srcPccRule == nil && targetPccRule == nil {
		return fmt.Errorf("No pccRule to apply tcData")
	}

	if applyTcID == "" && targetTcData == nil {
		logger.PduSessLog.Infof("No srcTcData and targetTcData. Nothing to do")
		return nil
	}

	//Set reference to traffic control data
	if applyTcID != "" {
		srcTcData, exist := smContext.TrafficControlPool[applyTcID]
		if exist {
			//TODO: Fix always choosing the first RouteToLocs as srcTraRouting
			srcTraRouting = srcTcData.RouteToLocs[0]
			//If no targetTcData, the default UpPathChgEvent will be the one in srcTcData
			upPathChgEvt = srcTcData.UpPathChgEvent
		} else {
			if targetTcData == nil {
				return fmt.Errorf("No this Traffic control data [%s] to remove", applyTcID)
			}
			//No srcTcData, get related info from SMContext
			srcTraRouting = models.RouteToLocation{
				Dnai: "", //TODO: Get the source DNAI
				RouteInfo: &models.RouteInformation{
					Ipv4Addr:   smContext.PDUAddress.String(),
					PortNumber: 2152, //TODO: Get the port from API
				},
			}
		}

		if targetTcData != nil {
			//TODO: Fix always choosing the first RouteToLocs as targetTraRouting
			targetTraRouting = targetTcData.RouteToLocs[0]
			//If targetTcData is available, change UpPathChgEvent to the one in targetTcData
			upPathChgEvt = targetTcData.UpPathChgEvent
		} else {
			//No targetTcData in decision, roll back to the original traffic routing
			targetTraRouting = models.RouteToLocation{
				Dnai: "", //TODO: Get the source DNAI
				RouteInfo: &models.RouteInformation{
					Ipv4Addr:   smContext.PDUAddress.String(),
					PortNumber: 2152, //TODO: Get the port from API
				},
			}
		}

		if !reflect.DeepEqual(srcTraRouting, targetTraRouting) {
			trChanged = true
		}
		if trChanged {
			//Send Notification to NEF/AF if UP path change type contains "EARLY"
			SendUpPathChgEventExposureNotification(upPathChgEvt, "EARLY",
				&srcTraRouting, &targetTraRouting)
		}

		if targetPccRule != nil {
			if err := applyDataPathWithTrafficControl(smContext, targetPccRule, &targetTraRouting); err != nil {
				return err
			}
			SendPFCPRule(smContext, targetPccRule.Datapath)
		}

		// Now our solution is to install new datapath first, and then remove old datapath
		if srcPccRule != nil && srcPccRule.Datapath != nil {
			removeDataPath(smContext, srcPccRule.Datapath)
			SendPFCPRule(smContext, srcPccRule.Datapath)
		}

		if trChanged {
			//Send Notification to NEF/AF if UP path change type contains "LATE"
			SendUpPathChgEventExposureNotification(upPathChgEvt, "LATE",
				&srcTraRouting, &targetTraRouting)
		}

		if targetTcData != nil {
			// Install traffic control data
			smContext.TrafficControlPool[applyTcID] = targetTcData
		} else { // Remove traffic control data
			delete(smContext.TrafficControlPool, applyTcID)
		}
	} else {
		return fmt.Errorf("Traffic control data ref invalid")
	}
	return nil
}

func applyDataPathWithTrafficControl(smContext *smf_context.SMContext, pccRule *smf_context.PCCRule,
	targetTraRouting *models.RouteToLocation) error {
	var routeProf factory.RouteProfile
	routeProfExist := false
	// specify N6 routing information
	if targetTraRouting.RouteProfId != "" {
		routeProf, routeProfExist = factory.UERoutingConfig.RouteProf[factory.RouteProfID(targetTraRouting.RouteProfId)]
		if !routeProfExist {
			return fmt.Errorf("Route Profile ID [%s] is not support", targetTraRouting.RouteProfId)
		}
	}
	for curDPNode := pccRule.Datapath.FirstDPNode; curDPNode != nil; curDPNode = curDPNode.Next() {
		if curDPNode.DownLinkTunnel != nil && curDPNode.DownLinkTunnel.PDR != nil {
			curDPNode.DownLinkTunnel.PDR.Precedence = uint32(pccRule.Precedence)
		}
		if curDPNode.UpLinkTunnel != nil && curDPNode.UpLinkTunnel.PDR != nil {
			curDPNode.UpLinkTunnel.PDR.Precedence = uint32(pccRule.Precedence)
		}
		if curDPNode.IsAnchorUPF() {
			curDLFAR := curDPNode.DownLinkTunnel.PDR.FAR
			if curDLFAR.ForwardingParameters == nil {
				curDLFAR.ForwardingParameters = new(smf_context.ForwardingParameters)
			}
			if routeProfExist {
				curDPNode.UpLinkTunnel.PDR.FAR.ForwardingParameters.ForwardingPolicyID = routeProf.ForwardingPolicyID
			}
			//TODO: Support the RouteInfo in targetTraRouting
			//TODO: Check the message is only presents one of RouteInfo or RouteProfId and sends failure message to the PCF
			// } else if routeInfo := targetTraRouting.RouteInfo; routeInfo != nil {
			// 	locToRouteIP := net.ParseIP(routeInfo.Ipv4Addr)
			// 	curDPNode.UpLinkTunnel.PDR.FAR.ForwardingParameters.OuterHeaderCreation = &pfcpType.OuterHeaderCreation{
			// 		OuterHeaderCreationDescription: pfcpType.OuterHeaderCreationUdpIpv4,
			// 		Ipv4Address:                    locToRouteIP,
			// 		PortNumber:                     uint16(routeInfo.PortNumber),
			// 	}
			// }
		}
		// get old TEID
		//TODO: remove this if RAN tunnel issue is fixed, because the AN tunnel is only one
		if curDPNode.IsANUPF() {
			curDPNode.UpLinkTunnel.PDR.PDI.LocalFTeid.Teid =
				smContext.Tunnel.DataPathPool.GetDefaultPath().FirstDPNode.GetUpLinkPDR().PDI.LocalFTeid.Teid
		}
	}
	return nil
}

func updatePccRuleDataPathFlowDescription(pccRule *smf_context.PCCRule,
	downlinkFlowDescription, uplinkFlowDescription string) {
	for curDPNode := pccRule.Datapath.FirstDPNode; curDPNode != nil; curDPNode = curDPNode.Next() {
		curDPNode.DownLinkTunnel.PDR.PDI.SDFFilter = &pfcpType.SDFFilter{
			Bid:                     false,
			Fl:                      false,
			Spi:                     false,
			Ttc:                     false,
			Fd:                      true,
			LengthOfFlowDescription: uint16(len(downlinkFlowDescription)),
			FlowDescription:         []byte(downlinkFlowDescription),
		}
		curDPNode.UpLinkTunnel.PDR.PDI.SDFFilter = &pfcpType.SDFFilter{
			Bid:                     false,
			Fl:                      false,
			Spi:                     false,
			Ttc:                     false,
			Fd:                      true,
			LengthOfFlowDescription: uint16(len(uplinkFlowDescription)),
			FlowDescription:         []byte(uplinkFlowDescription),
		}
	}
}

func getUplinkFlowDescription(downlinkFlowDesc string) string {
	var err error
	uplinkFlowDescription := ""
	uplinkIPFilterRule := flowdesc.NewIPFilterRule()

	if err = flowdesc.Decode(downlinkFlowDesc, uplinkIPFilterRule); err == nil {
		uplinkIPFilterRule.SwapSourceAndDestination()
		if uplinkFlowDescription, err = flowdesc.Encode(uplinkIPFilterRule); err != nil {
			return ""
		}
	}
	return uplinkFlowDescription
}
