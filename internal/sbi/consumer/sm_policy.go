package consumer

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"bitbucket.org/free5gc-team/nas/nasConvert"
	"bitbucket.org/free5gc-team/nas/nasMessage"
	"bitbucket.org/free5gc-team/nas/nasType"
	"bitbucket.org/free5gc-team/openapi/models"
	"bitbucket.org/free5gc-team/pfcp"
	smf_context "bitbucket.org/free5gc-team/smf/internal/context"
	"bitbucket.org/free5gc-team/smf/internal/logger"
	"bitbucket.org/free5gc-team/util/flowdesc"
)

// SendSMPolicyAssociationCreate create the session management association to the PCF
func SendSMPolicyAssociationCreate(smContext *smf_context.SMContext) (string, *models.SmPolicyDecision, error) {
	if smContext.SMPolicyClient == nil {
		return "", nil, errors.Errorf("smContext not selected PCF")
	}

	smPolicyData := models.SmPolicyContextData{}

	smPolicyData.Supi = smContext.Supi
	smPolicyData.PduSessionId = smContext.PDUSessionID
	smPolicyData.NotificationUri = fmt.Sprintf("%s://%s:%d/nsmf-callback/sm-policies/%s",
		smf_context.GetSelf().URIScheme,
		smf_context.GetSelf().RegisterIPv4,
		smf_context.GetSelf().SBIPort,
		smContext.Ref,
	)
	smPolicyData.Dnn = smContext.Dnn
	smPolicyData.PduSessionType = nasConvert.PDUSessionTypeToModels(smContext.SelectedPDUSessionType)
	smPolicyData.AccessType = smContext.AnType
	smPolicyData.RatType = smContext.RatType
	smPolicyData.Ipv4Address = smContext.PDUAddress.To4().String()
	smPolicyData.SubsSessAmbr = smContext.DnnConfiguration.SessionAmbr
	smPolicyData.SubsDefQos = smContext.DnnConfiguration.Var5gQosProfile
	smPolicyData.SliceInfo = smContext.SNssai
	smPolicyData.ServingNetwork = &models.NetworkId{
		Mcc: smContext.ServingNetwork.Mcc,
		Mnc: smContext.ServingNetwork.Mnc,
	}
	smPolicyData.SuppFeat = "F"

	var smPolicyID string
	var smPolicyDecision *models.SmPolicyDecision
	smPolicyDecisionFromPCF, httpRsp, err := smContext.SMPolicyClient.DefaultApi.
		SmPoliciesPost(context.Background(), smPolicyData)
	defer func() {
		if httpRsp != nil {
			if closeErr := httpRsp.Body.Close(); closeErr != nil {
				logger.PduSessLog.Errorf("rsp body close err: %v", closeErr)
			}
		}
	}()
	if err != nil {
		return "", nil, err
	}

	smPolicyDecision = &smPolicyDecisionFromPCF
	loc := httpRsp.Header.Get("Location")
	if smPolicyID = extractSMPolicyIDFromLocation(loc); len(smPolicyID) == 0 {
		return "", nil, fmt.Errorf("SMPolicy ID parse failed")
	}
	return smPolicyID, smPolicyDecision, nil
}

var smPolicyRegexp = regexp.MustCompile(`http[s]?\://.*/npcf-smpolicycontrol/v\d+/sm-policies/(.*)`)

func extractSMPolicyIDFromLocation(location string) string {
	match := smPolicyRegexp.FindStringSubmatch(location)
	if len(match) > 1 {
		return match[1]
	}
	// not match submatch
	return ""
}

func SendSMPolicyAssociationUpdateByUERequestModification(
	smContext *smf_context.SMContext,
	qosRules nasType.QoSRules, qosFlowDescs nasType.QoSFlowDescs,
) (*models.SmPolicyDecision, error) {
	updateSMPolicy := models.SmPolicyUpdateContextData{}

	updateSMPolicy.RepPolicyCtrlReqTriggers = []models.PolicyControlRequestTrigger{
		models.PolicyControlRequestTrigger_RES_MO_RE,
	}

	// UE SHOULD only create ONE QoS Flow in a request (TS 24.501 6.4.2.2)
	rule := qosRules[0]
	flowDesc := qosFlowDescs[0]

	var ruleOp models.RuleOperation
	switch rule.Operation {
	case nasType.OperationCodeCreateNewQoSRule:
		ruleOp = models.RuleOperation_CREATE_PCC_RULE
	case nasType.OperationCodeDeleteExistingQoSRule:
		ruleOp = models.RuleOperation_DELETE_PCC_RULE
	case nasType.OperationCodeModifyExistingQoSRuleAndAddPacketFilters:
		ruleOp = models.RuleOperation_MODIFY_PCC_RULE_AND_ADD_PACKET_FILTERS
	case nasType.OperationCodeModifyExistingQoSRuleAndDeletePacketFilters:
		ruleOp = models.RuleOperation_MODIFY_PCC_RULE_AND_DELETE_PACKET_FILTERS
	case nasType.OperationCodeModifyExistingQoSRuleAndReplaceAllPacketFilters:
		ruleOp = models.RuleOperation_MODIFY_PCC_RULE_AND_REPLACE_PACKET_FILTERS
	case nasType.OperationCodeModifyExistingQoSRuleWithoutModifyingPacketFilters:
		ruleOp = models.RuleOperation_MODIFY_PCC_RULE_WITHOUT_MODIFY_PACKET_FILTERS
	default:
		return nil, errors.New("QoS Rule Operation Unknown")
	}

	ueInitResReq := &models.UeInitiatedResourceRequest{}
	ueInitResReq.RuleOp = ruleOp
	ueInitResReq.Precedence = int32(rule.Precedence)
	ueInitResReq.ReqQos = new(models.RequestedQos)

	for _, parameter := range flowDesc.Parameters {
		switch parameter.Identifier() {
		case nasType.ParameterIdentifier5QI:
			para5Qi := parameter.(*nasType.QoSFlow5QI)
			ueInitResReq.ReqQos.Var5qi = int32(para5Qi.FiveQI)
		case nasType.ParameterIdentifierGFBRUplink:
			paraGFBRUplink := parameter.(*nasType.QoSFlowGFBRUplink)
			ueInitResReq.ReqQos.GbrUl = nasBitRateToString(paraGFBRUplink.Value, paraGFBRUplink.Unit)
		case nasType.ParameterIdentifierGFBRDownlink:
			paraGFBRUplink := parameter.(*nasType.QoSFlowGFBRUplink)
			ueInitResReq.ReqQos.GbrUl = nasBitRateToString(paraGFBRUplink.Value, paraGFBRUplink.Unit)
		}
	}

	updateSMPolicy.UeInitResReq = ueInitResReq

	for _, pf := range rule.PacketFilterList {
		if PackFiltInfo, err := buildPktFilterInfo(pf); err != nil {
			smContext.Log.Warning("Build PackFiltInfo failed", err)
			continue
		} else {
			updateSMPolicy.UeInitResReq.PackFiltInfo = append(updateSMPolicy.UeInitResReq.PackFiltInfo, *PackFiltInfo)
		}
	}
	var smPolicyDecision *models.SmPolicyDecision
	smPolicyDecisionFromPCF, rsp, err := smContext.SMPolicyClient.
		DefaultApi.SmPoliciesSmPolicyIdUpdatePost(context.TODO(), smContext.SMPolicyID, updateSMPolicy)
	defer func() {
		if rsp != nil {
			if closeErr := rsp.Body.Close(); closeErr != nil {
				logger.PduSessLog.Errorf("rsp body close err: %v", closeErr)
			}
		}
	}()
	if err != nil {
		return nil, fmt.Errorf("update sm policy [%s] association failed: %s", smContext.SMPolicyID, err)
	}
	smPolicyDecision = &smPolicyDecisionFromPCF
	return smPolicyDecision, nil
}

func nasBitRateToString(value uint16, unit nasType.QoSFlowBitRateUnit) string {
	var base int
	var unitStr string
	switch unit {
	case nasType.QoSFlowBitRateUnit1Kbps:
		base = 1
		unitStr = "Kbps"
	case nasType.QoSFlowBitRateUnit4Kbps:
		base = 4
		unitStr = "Kbps"
	case nasType.QoSFlowBitRateUnit16Kbps:
		base = 16
		unitStr = "Kbps"
	case nasType.QoSFlowBitRateUnit64Kbps:
		base = 64
		unitStr = "Kbps"
	case nasType.QoSFlowBitRateUnit256Kbps:
		base = 256
		unitStr = "Kbps"
	case nasType.QoSFlowBitRateUnit1Mbps:
		base = 1
		unitStr = "Mbps"
	case nasType.QoSFlowBitRateUnit4Mbps:
		base = 4
		unitStr = "Mbps"
	case nasType.QoSFlowBitRateUnit16Mbps:
		base = 16
		unitStr = "Mbps"
	case nasType.QoSFlowBitRateUnit64Mbps:
		base = 64
		unitStr = "Mbps"
	case nasType.QoSFlowBitRateUnit256Mbps:
		base = 256
		unitStr = "Mbps"
	case nasType.QoSFlowBitRateUnit1Gbps:
		base = 1
		unitStr = "Gbps"
	case nasType.QoSFlowBitRateUnit4Gbps:
		base = 4
		unitStr = "Gbps"
	case nasType.QoSFlowBitRateUnit16Gbps:
		base = 16
		unitStr = "Gbps"
	case nasType.QoSFlowBitRateUnit64Gbps:
		base = 64
		unitStr = "Gbps"
	case nasType.QoSFlowBitRateUnit256Gbps:
		base = 256
		unitStr = "Gbps"
	case nasType.QoSFlowBitRateUnit1Tbps:
		base = 1
		unitStr = "Tbps"
	case nasType.QoSFlowBitRateUnit4Tbps:
		base = 4
		unitStr = "Tbps"
	case nasType.QoSFlowBitRateUnit16Tbps:
		base = 16
		unitStr = "Tbps"
	case nasType.QoSFlowBitRateUnit64Tbps:
		base = 64
		unitStr = "Tbps"
	case nasType.QoSFlowBitRateUnit256Tbps:
		base = 256
		unitStr = "Tbps"
	case nasType.QoSFlowBitRateUnit1Pbps:
		base = 1
		unitStr = "Pbps"
	case nasType.QoSFlowBitRateUnit4Pbps:
		base = 4
		unitStr = "Pbps"
	case nasType.QoSFlowBitRateUnit16Pbps:
		base = 16
		unitStr = "Pbps"
	case nasType.QoSFlowBitRateUnit64Pbps:
		base = 64
		unitStr = "Pbps"
	case nasType.QoSFlowBitRateUnit256Pbps:
		base = 256
		unitStr = "Pbps"
	default:
		base = 1
		unitStr = "Kbps"
	}

	return fmt.Sprintf("%d %s", base*int(value), unitStr)
}

func StringToNasBitRate(str string) (uint16, nasType.QoSFlowBitRateUnit, error) {
	strSegment := strings.Split(str, " ")

	var unit nasType.QoSFlowBitRateUnit
	switch strSegment[1] {
	case "Kbps":
		unit = nasType.QoSFlowBitRateUnit1Kbps
	case "Mbps":
		unit = nasType.QoSFlowBitRateUnit1Mbps
	case "Gbps":
		unit = nasType.QoSFlowBitRateUnit1Gbps
	case "Tbps":
		unit = nasType.QoSFlowBitRateUnit1Tbps
	case "Pbps":
		unit = nasType.QoSFlowBitRateUnit1Pbps
	default:
		unit = nasType.QoSFlowBitRateUnit1Kbps
	}

	if value, err := strconv.Atoi(strSegment[0]); err != nil {
		return 0, 0, err
	} else {
		return uint16(value), unit, err
	}
}

func buildPktFilterInfo(pf nasType.PacketFilter) (*models.PacketFilterInfo, error) {
	pfInfo := &models.PacketFilterInfo{}

	switch pf.Direction {
	case nasType.PacketFilterDirectionDownlink:
		pfInfo.FlowDirection = models.FlowDirection_DOWNLINK
	case nasType.PacketFilterDirectionUplink:
		pfInfo.FlowDirection = models.FlowDirection_UPLINK
	case nasType.PacketFilterDirectionBidirectional:
		pfInfo.FlowDirection = models.FlowDirection_BIDIRECTIONAL
	default:
		pfInfo.FlowDirection = models.FlowDirection_UNSPECIFIED
	}

	const ProtocolNumberAny = 0xfc
	packetFilter := &flowdesc.IPFilterRule{
		Action: "permit",
		Dir:    "out",
		Proto:  ProtocolNumberAny,
	}

	for _, component := range pf.Components {
		switch component.Type() {
		case nasType.PacketFilterComponentTypeIPv4RemoteAddress:
			ipv4Remote := component.(*nasType.PacketFilterIPv4RemoteAddress)
			remoteIPnet := net.IPNet{
				IP:   ipv4Remote.Address,
				Mask: ipv4Remote.Mask,
			}
			packetFilter.Src = remoteIPnet.String()
		case nasType.PacketFilterComponentTypeIPv4LocalAddress:
			ipv4Local := component.(*nasType.PacketFilterIPv4LocalAddress)
			localIPnet := net.IPNet{
				IP:   ipv4Local.Address,
				Mask: ipv4Local.Mask,
			}
			packetFilter.Dst = localIPnet.String()
		case nasType.PacketFilterComponentTypeProtocolIdentifierOrNextHeader:
			protoNumber := component.(*nasType.PacketFilterProtocolIdentifier)
			packetFilter.Proto = protoNumber.Value

		case nasType.PacketFilterComponentTypeSingleLocalPort:
			localPort := component.(*nasType.PacketFilterSingleLocalPort)
			packetFilter.DstPorts = append(packetFilter.DstPorts, flowdesc.PortRange{
				Start: localPort.Value,
				End:   localPort.Value,
			})
		case nasType.PacketFilterComponentTypeLocalPortRange:
			localPortRange := component.(*nasType.PacketFilterLocalPortRange)
			packetFilter.DstPorts = append(packetFilter.DstPorts, flowdesc.PortRange{
				Start: localPortRange.LowLimit,
				End:   localPortRange.HighLimit,
			})
		case nasType.PacketFilterComponentTypeSingleRemotePort:
			remotePort := component.(*nasType.PacketFilterSingleRemotePort)
			packetFilter.SrcPorts = append(packetFilter.SrcPorts, flowdesc.PortRange{
				Start: remotePort.Value,
				End:   remotePort.Value,
			})
		case nasType.PacketFilterComponentTypeRemotePortRange:
			remotePortRange := component.(*nasType.PacketFilterRemotePortRange)
			packetFilter.SrcPorts = append(packetFilter.SrcPorts, flowdesc.PortRange{
				Start: remotePortRange.LowLimit,
				End:   remotePortRange.HighLimit,
			})
		case nasType.PacketFilterComponentTypeSecurityParameterIndex:
			securityParameter := component.(*nasType.PacketFilterSecurityParameterIndex)
			pfInfo.Spi = fmt.Sprintf("%04x", securityParameter.Index)
		case nasType.PacketFilterComponentTypeTypeOfServiceOrTrafficClass:
			serviceClass := component.(*nasType.PacketFilterServiceClass)
			pfInfo.TosTrafficClass = fmt.Sprintf("%x%x", serviceClass.Class, serviceClass.Mask)
		case nasType.PacketFilterComponentTypeFlowLabel:
			flowLabel := component.(*nasType.PacketFilterFlowLabel)
			pfInfo.FlowLabel = fmt.Sprintf("%03x", flowLabel.Label)
		}
	}

	if desc, err := flowdesc.Encode(packetFilter); err != nil {
		return nil, err
	} else {
		pfInfo.PackFiltCont = desc
	}
	// according TS 29.212 IPFilterRule cannot use [options]
	return pfInfo, nil
}

func SendSMPolicyAssociationTermination(smContext *smf_context.SMContext) error {
	if smContext.SMPolicyClient == nil {
		return errors.Errorf("smContext not selected PCF")
	}

	rsp, err := smContext.SMPolicyClient.DefaultApi.SmPoliciesSmPolicyIdDeletePost(
		context.Background(), smContext.SMPolicyID, models.SmPolicyDeleteData{})
	defer func() {
		if rsp != nil {
			if closeErr := rsp.Body.Close(); closeErr != nil {
				logger.PduSessLog.Errorf("rsp body close err: %v", closeErr)
			}
		}
	}()
	if err != nil {
		return fmt.Errorf("SM Policy termination failed: %v", err)
	}
	return nil
}

// SMF initiated SM Policy Association Modification
// Npcf_SMPolicyControl_Update request
func SendSMPolicyAssociationUpdateByReportTSNInformation(
	smContext *smf_context.SMContext,
	req *nasMessage.PDUSessionEstablishmentRequest,
	pdu_session_id uint8) (*models.SmPolicyDecision, error) {
	updateSMPolicy := models.SmPolicyUpdateContextData{}

	// 29.512 "TimeSensitiveNetworking" or "TimeSensitiveCommunication" feature is supported
	updateSMPolicy.RepPolicyCtrlReqTriggers = []models.PolicyControlRequestTrigger{
		models.PolicyControlRequestTrigger_TSN_BRIDGE_INFO,
	}

	// 29.512 5.6.2.41 Transports TSN bridge information.
	tsnbridgeinfo := &models.TsnBridgeInformation{}

	// mac Dstt port from [6]uint8 to string
	if smContext.SelectedPDUSessionType == nasMessage.PDUSessionTypeEthernet {
		tsnbridgeinfo.DsttAddr = hex.EncodeToString(req.DSTTEthernetportMACaddress.Octet[:])
	}

	// TODO: UEDSTTresidencetime tpye should be Uinteger
	if req.UEDSTTresidencetime != nil {
		tsnbridgeinfo.DsttResidTime = req.UEDSTTresidencetime.Octet
	}

	port_pair_info, exist := smContext.BridgeInfo[pdu_session_id]
	if !exist {
		return nil, errors.New("Cannot find port pair information")
	}
	tsnbridgeinfo.DsttPortNum = port_pair_info.DSTTPortNumber
	tsnbridgeinfo.BridgeId = port_pair_info.BridgeId
	updateSMPolicy.TsnBridgeInfo = tsnbridgeinfo

	logger.ConsumerLog.Debugf("\ntsnbridgeinfo.BridgeId: [%d]\ntsnbridgeinfo.DsttPortNum: [%d]\ntsnbridgeinfo.DsttAddr: [%s]\ntsnbridgeinfo.DsttResidTime: [%x]\n",
		tsnbridgeinfo.BridgeId,
		tsnbridgeinfo.DsttPortNum,
		tsnbridgeinfo.DsttAddr,
		tsnbridgeinfo.DsttResidTime,
	)

	// // 29.512 5.6.2.47 Transports TSC bridge management information(user plane node management information)
	// umic, exist := smContext.Nwtt_UMIC[pdu_session_id]
	// if exist {
	// 	updateSMPolicy.TsnBridgeManCont = &models.BridgeManagementContainer{
	// 		BridgeManCont: umic,
	// 	}
	// } else {
	// 	logger.ConsumerLog.Debugln("No Nwtt_UMIC in smContext")
	// }

	// 29.512 5.6.2.45 Transports TSN port management information for the DS-TT port.
	updateSMPolicy.TsnPortManContDstt = &models.PortManagementContainer{
		PortManCont: req.PortManagementInformationContainer.Buffer,
		PortNum:     port_pair_info.DSTTPortNumber,
	}

	// // Transports TSN port management information for the NW-TTs port.
	// for portNum, portManCont := range smContext.Nwtt_PMIC {
	// 	pmic := models.PortManagementContainer{
	// 		PortManCont: portManCont,
	// 		PortNum:     portNum,
	// 	}
	// 	logger.ConsumerLog.Debugf("Transport PortNum: [%d], PortManCont: [%x]\n", portNum, portManCont)
	// 	updateSMPolicy.TsnPortManContNwtts = append(updateSMPolicy.TsnPortManContNwtts, pmic)
	// }

	var smPolicyDecision *models.SmPolicyDecision
	if smPolicyDecisionFromPCF, _, err := smContext.SMPolicyClient.
		DefaultApi.SmPoliciesSmPolicyIdUpdatePost(context.TODO(), smContext.SMPolicyID, updateSMPolicy); err != nil {
		return nil, fmt.Errorf("update sm policy [%s] association failed: %s", smContext.SMPolicyID, err)
	} else {
		smPolicyDecision = &smPolicyDecisionFromPCF
	}

	return smPolicyDecision, nil
}

// Npcf_SMPolicyControl_Update request
func SendSMPolicyAssociationUpdateByReportTSNInformation_NWTT(
	smContext *smf_context.SMContext,
	TMIReport []*pfcp.TSCManagementInformation,
	pdu_session_id uint8) (*models.SmPolicyDecision, error) {
	updateSMPolicy := models.SmPolicyUpdateContextData{}

	updateSMPolicy.RepPolicyCtrlReqTriggers = []models.PolicyControlRequestTrigger{
		models.PolicyControlRequestTrigger_TSN_BRIDGE_INFO,
	}

	bridgeinfo, exist := smContext.BridgeInfo[pdu_session_id]
	if !exist {
		logger.ConsumerLog.Errorln("bridgeinfo doesn't exist")
		return nil, nil
	}

	for _, report := range TMIReport {
		if report.NWTTPortNumber != nil && report.NWTTPortNumber.PortNumberValue != 0 {
			// Transports TSN port management information for the NW-TTs port.
			logger.ConsumerLog.Infoln("Received PMIC: [", report.PortManagementInformationContainer.PortManagementInformation, "] for port",
				report.NWTTPortNumber.PortNumberValue)
			bridgeinfo.NWTTPortNumber = report.NWTTPortNumber.PortNumberValue
			smContext.Nwtt_PMIC[bridgeinfo.NWTTPortNumber] = report.PortManagementInformationContainer.PortManagementInformation

			pmic := models.PortManagementContainer{
				PortManCont: smContext.Nwtt_PMIC[bridgeinfo.NWTTPortNumber],
				PortNum:     bridgeinfo.NWTTPortNumber,
			}
			updateSMPolicy.TsnPortManContNwtts = append(updateSMPolicy.TsnPortManContNwtts, pmic)

		} else if report.BridgeManagementInformationContainer != nil {
			// 29.512 5.6.2.47 Transports TSC bridge management information(user plane node management information)
			logger.ConsumerLog.Infoln("Received UMIC: ", report.BridgeManagementInformationContainer.BridgeManagementInformation)
			smContext.Nwtt_UMIC[pdu_session_id] = report.BridgeManagementInformationContainer.BridgeManagementInformation

			updateSMPolicy.TsnBridgeManCont = &models.BridgeManagementContainer{
				BridgeManCont: smContext.Nwtt_UMIC[pdu_session_id],
			}

		} else {
			logger.ConsumerLog.Errorln("TMIReport error")
			return nil, nil
		}
	}

	var smPolicyDecision *models.SmPolicyDecision
	if smPolicyDecisionFromPCF, _, err := smContext.SMPolicyClient.
		DefaultApi.SmPoliciesSmPolicyIdUpdatePost(context.TODO(), smContext.SMPolicyID, updateSMPolicy); err != nil {
		return nil, fmt.Errorf("update sm policy [%s] association failed: %s", smContext.SMPolicyID, err)
	} else {
		smPolicyDecision = &smPolicyDecisionFromPCF
	}

	return smPolicyDecision, nil
}

// Npcf_SMPolicyControl_Update request
func SendSMPolicyAssociationUpdateByReportTSNInformation_DSTT(
	smContext *smf_context.SMContext,
	req *nasMessage.PDUSessionModificationComplete) (*models.SmPolicyDecision, error) {
	updateSMPolicy := models.SmPolicyUpdateContextData{}

	updateSMPolicy.RepPolicyCtrlReqTriggers = []models.PolicyControlRequestTrigger{
		models.PolicyControlRequestTrigger_TSN_BRIDGE_INFO,
	}

	portnum, exist := smContext.Dstt_portnum[uint8(req.GetPDUSessionID())]
	if !exist {
		return nil, errors.New("Cannot find Dstt port number")
	}

	if req.PortManagementInformationContainer == nil {
		return nil, errors.New("Cannot find PMIC")
	}
	updateSMPolicy.TsnPortManContDstt = &models.PortManagementContainer{
		PortManCont: req.PortManagementInformationContainer.Buffer,
		PortNum:     portnum,
	}

	var smPolicyDecision *models.SmPolicyDecision
	if smPolicyDecisionFromPCF, _, err := smContext.SMPolicyClient.
		DefaultApi.SmPoliciesSmPolicyIdUpdatePost(context.TODO(), smContext.SMPolicyID, updateSMPolicy); err != nil {
		return nil, fmt.Errorf("update sm policy [%s] association failed: %s", smContext.SMPolicyID, err)
	} else {
		smPolicyDecision = &smPolicyDecisionFromPCF
	}

	return smPolicyDecision, nil
}
