package producer

import (
	"context"
	"free5gc/lib/flowdesc"
	"free5gc/lib/http_wrapper"
	"free5gc/lib/openapi/Nsmf_EventExposure"
	"free5gc/lib/openapi/models"
	"free5gc/lib/pfcp/pfcpType"
	smf_context "free5gc/src/smf/context"
	"free5gc/src/smf/factory"
	"free5gc/src/smf/logger"
	"net/http"
	"reflect"
	"strings"
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

	if smContext.SMContextState != smf_context.Active {
		//Wait till the state becomes Active again
		//TODO: implement waiting in concurrent architecture
		logger.PduSessLog.Infoln("The SMContext State should be Active State")
		logger.PduSessLog.Infoln("SMContext state: ", smContext.SMContextState.String())
	}

	//TODO: Response data type -
	//[200 OK] UeCampingRep
	//[200 OK] array(PartialSuccessReport)
	//[400 Bad Request] ErrorReport
	httpResponse := http_wrapper.NewResponse(http.StatusNoContent, nil, nil)
	if err := ApplySmPolicyFromDecision(smContext, decision); err != nil {
		//TODO: Fill the error body
		httpResponse.Status = http.StatusBadRequest
	}

	return httpResponse
}

func SendUpPathChgEventExposureNotification(
	chgEvent *models.UpPathChgEvent,
	chgType string, sourceTR,
	targetTR *models.RouteToLocation) {
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

	if chgEvent.NotificationUri != "" && strings.Contains(string(chgEvent.DnaiChgType), chgType) {
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

func ApplySmPolicyFromDecision(smContext *smf_context.SMContext, decision *models.SmPolicyDecision) error {
	logger.PduSessLog.Traceln("In ApplySmPolicyFromDecision")
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

	pccRuleUpdated := false
	for id, pccRuleModel := range decision.PccRules {
		pccRule, exist := smContext.PCCRules[id]
		//TODO: Change PccRules map[string]PccRule to map[string]*PccRule

		pccRulePtr := &pccRuleModel
		if pccRulePtr == nil {
			logger.PduSessLog.Infof("Remove PCCRule[%s]", id)
			if !exist {
				logger.PduSessLog.Errorf("pcc rule [%s] not exist", id)
				continue
			}

		} else {
			if exist {
				logger.PduSessLog.Infof("Modify PCCRule[%s]", id)
			} else {
				logger.PduSessLog.Infof("Install PCCRule[%s]", id)
			}

			newPccRule := smf_context.NewPCCRuleFromModel(pccRuleModel)
			upfSelectionParams := &smf_context.UPFSelectionParams{
				Dnn: smContext.Dnn,
				SNssai: &smf_context.SNssai{
					Sst: smContext.Snssai.Sst,
					Sd:  smContext.Snssai.Sd,
				},
			}
			// Create data traffic for the new PCC Rule
			createdUpPath := smf_context.GetUserPlaneInformation().GetDefaultUserPlanePathByDNN(upfSelectionParams)
			createdDataPath := smf_context.GenerateDataPath(createdUpPath, smContext)
			createdDataPath.ActivateTunnelAndPDR(smContext)
			smContext.Tunnel.AddDataPath(createdDataPath)

			updatePccRule, updateTcData, trChanged := false, false, false
			var sourceTraRouting, targetTraRouting models.RouteToLocation
			var tcModel models.TrafficControlData

			if appID := newPccRule.AppID; appID != "" {
				var matchedPFD *factory.PfdDataForApp
				for _, pfdDataForApp := range factory.UERoutingConfig.PfdDatas {
					if pfdDataForApp.AppID == appID {
						matchedPFD = pfdDataForApp
						break
					}
				}

				if matchedPFD != nil && matchedPFD.Pfds != nil && matchedPFD.Pfds[0].FlowDescriptions != nil {
					flowDescConfig := matchedPFD.Pfds[0].FlowDescriptions[0]
					uplinkIPFilterRule := flowdesc.NewIPFilterRule()

					if err := flowdesc.Decode(flowDescConfig, uplinkIPFilterRule); err != nil {
						logger.PduSessLog.Error(err)
					} else {
						uplinkIPFilterRule.SwapSourceAndDestination()
						var uplinkFlowDescription string
						if ret, err := flowdesc.Encode(uplinkIPFilterRule); err != nil {
							logger.PduSessLog.Error(err)
						} else {
							uplinkFlowDescription = ret
						}

						for curDPNode := createdDataPath.FirstDPNode; curDPNode != nil; curDPNode = curDPNode.Next() {

							curDPNode.DownLinkTunnel.PDR.PDI.SDFFilter = &pfcpType.SDFFilter{
								Bid:                     false,
								Fl:                      false,
								Spi:                     false,
								Ttc:                     false,
								Fd:                      true,
								LengthOfFlowDescription: uint16(len(flowDescConfig)),
								FlowDescription:         []byte(flowDescConfig),
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
				} else {
					logger.PduSessLog.Errorf("No PFD matched for Aplicationp ID [%s]", appID)
					continue
				}
			}

			//Set reference to traffic control data
			if len(pccRuleModel.RefTcData) != 0 && pccRuleModel.RefTcData[0] != "" {
				refTcID := pccRuleModel.RefTcData[0]
				tcModel = *decision.TraffContDecs[refTcID]
				newTcData := smf_context.NewTrafficControlDataFromModel(&tcModel)

				routeToLoc := tcModel.RouteToLocs[0]

				for curDPNode := createdDataPath.FirstDPNode; curDPNode != nil; curDPNode = curDPNode.Next() {
					if curDPNode.DownLinkTunnel != nil && curDPNode.DownLinkTunnel.PDR != nil {
						curDPNode.DownLinkTunnel.PDR.Precedence = uint32(newPccRule.Precedence)
					}
					if curDPNode.UpLinkTunnel != nil && curDPNode.UpLinkTunnel.PDR != nil {
						curDPNode.UpLinkTunnel.PDR.Precedence = uint32(newPccRule.Precedence)
					}
					if curDPNode.IsAnchorUPF() {
						curDLFAR := curDPNode.DownLinkTunnel.PDR.FAR
						if curDLFAR.ForwardingParameters == nil {
							curDLFAR.ForwardingParameters = new(smf_context.ForwardingParameters)
						}
						// specify N6 routing information
						if routeToLoc.RouteProfId != "" {
							routeProf, exist := factory.UERoutingConfig.RouteProf[factory.RouteProfID(routeToLoc.RouteProfId)]
							if exist {
								curDPNode.UpLinkTunnel.PDR.FAR.ForwardingParameters.ForwardingPolicyID = routeProf.ForwardingPolicyID
							} else {
								logger.PduSessLog.Errorf("Route Profile ID [%s] is not support", routeToLoc.RouteProfId)
							}
						}
						//TODO: Support the RouteInfo in routeToLoc
						//TODO: Check the message is only presents one of RouteInfo or RouteProfId and sends failure message to the PCF
						// } else if routeInfo := routeToLoc.RouteInfo; routeInfo != nil {
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

				//TODO: Fix always choosing the first RouteToLocs as targetTraRouting
				targetTraRouting = newTcData.RouteToLocs[0]

				sourceTcData, exist := smContext.TrafficControlPool[refTcID]
				if exist {
					//TODO: Fix always choosing the first RouteToLocs as sourceTraRouting
					sourceTraRouting = sourceTcData.RouteToLocs[0]
					if !reflect.DeepEqual(sourceTraRouting, targetTraRouting) {
						trChanged, updateTcData, updatePccRule = true, true, true
					} else if !reflect.DeepEqual(sourceTcData, newTcData) {
						updateTcData, updatePccRule = true, true
					}
				} else { //No sourceTcData, get related info from SMContext
					//TODO: Get the source DNAI
					sourceTraRouting.Dnai = ""
					sourceTraRouting.RouteInfo = new(models.RouteInformation)
					sourceTraRouting.RouteInfo.Ipv4Addr = smContext.PDUAddress.String()
					//TODO: Get the port from API
					sourceTraRouting.RouteInfo.PortNumber = 2152
					trChanged, updateTcData, updatePccRule = true, true, true
				}

				if updateTcData {
					newPccRule.SetRefTrafficControlData(newTcData)
					smContext.TrafficControlPool[refTcID] = newTcData
				}
			}
			if !updatePccRule && !reflect.DeepEqual(pccRule, newPccRule) {
				updatePccRule = true
			}
			if trChanged {
				//Send Notification to NEF/AF if UP path change type contains "EARLY"
				SendUpPathChgEventExposureNotification(tcModel.UpPathChgEvent, "EARLY", &sourceTraRouting, &targetTraRouting)
			}
			if updatePccRule {
				smContext.PCCRules[id] = newPccRule
				//TODO: Update to UPF
			}

			SendPFCPRule(smContext, createdDataPath)

			if trChanged {
				//Send Notification to NEF/AF if UP path change type contains "LATE"
				SendUpPathChgEventExposureNotification(tcModel.UpPathChgEvent, "LATE", &sourceTraRouting, &targetTraRouting)
			}
			pccRuleUpdated = true
		}
	}

	var err error
	if pccRuleUpdated {
		smContext.SMContextState = smf_context.Active
	} else {
		//TODO: Follow spec to return the error reason
		logger.PduSessLog.Traceln("No PCC rule updated")
	}
	logger.PduSessLog.Traceln("End of ApplySmPolicyFromDecision")
	return err
}
