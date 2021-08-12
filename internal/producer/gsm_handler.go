package producer

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"bitbucket.org/free5GC/smf/internal/consumer"
	"bitbucket.org/free5GC/smf/internal/logger"
	"bitbucket.org/free5gc-team/nas"
	"bitbucket.org/free5gc-team/nas/nasConvert"
	"bitbucket.org/free5gc-team/nas/nasMessage"
	"bitbucket.org/free5gc-team/nas/nasType"
	"bitbucket.org/free5gc-team/openapi/models"
	smf_context "bitbucket.org/free5gc-team/smf/internal/context"
	"bitbucket.org/free5gc-team/util/flowdesc"
)

func HandlePDUSessionEstablishmentRequest(
	smCtx *smf_context.SMContext, req *nasMessage.PDUSessionEstablishmentRequest) {
	// Retrieve PDUSessionID
	smCtx.PDUSessionID = int32(req.PDUSessionID.GetPDUSessionID())
	logger.GsmLog.Infoln("In HandlePDUSessionEstablishmentRequest")

	// Retrieve PTI (Procedure transaction identity)
	smCtx.Pti = req.GetPTI()

	// Handle PDUSessionType
	if req.PDUSessionType != nil {
		requestedPDUSessionType := req.PDUSessionType.GetPDUSessionTypeValue()
		if err := smCtx.IsAllowedPDUSessionType(requestedPDUSessionType); err != nil {
			logger.CtxLog.Errorf("%s", err)
			return
		}
	} else {
		// Set to default supported PDU Session Type
		switch smf_context.SMF_Self().SupportedPDUSessionType {
		case "IPv4":
			smCtx.SelectedPDUSessionType = nasMessage.PDUSessionTypeIPv4
		case "IPv6":
			smCtx.SelectedPDUSessionType = nasMessage.PDUSessionTypeIPv6
		case "IPv4v6":
			smCtx.SelectedPDUSessionType = nasMessage.PDUSessionTypeIPv4IPv6
		case "Ethernet":
			smCtx.SelectedPDUSessionType = nasMessage.PDUSessionTypeEthernet
		default:
			smCtx.SelectedPDUSessionType = nasMessage.PDUSessionTypeIPv4
		}
	}

	if req.ExtendedProtocolConfigurationOptions != nil {
		EPCOContents := req.ExtendedProtocolConfigurationOptions.GetExtendedProtocolConfigurationOptionsContents()
		protocolConfigurationOptions := nasConvert.NewProtocolConfigurationOptions()
		unmarshalErr := protocolConfigurationOptions.UnMarshal(EPCOContents)
		if unmarshalErr != nil {
			logger.GsmLog.Errorf("Parsing PCO failed: %s", unmarshalErr)
		}
		logger.GsmLog.Infoln("Protocol Configuration Options")
		logger.GsmLog.Infoln(protocolConfigurationOptions)

		for _, container := range protocolConfigurationOptions.ProtocolOrContainerList {
			logger.GsmLog.Traceln("Container ID: ", container.ProtocolOrContainerID)
			logger.GsmLog.Traceln("Container Length: ", container.LengthOfContents)
			switch container.ProtocolOrContainerID {
			case nasMessage.PCSCFIPv6AddressRequestUL:
				logger.GsmLog.Infoln("Didn't Implement container type PCSCFIPv6AddressRequestUL")
			case nasMessage.IMCNSubsystemSignalingFlagUL:
				logger.GsmLog.Infoln("Didn't Implement container type IMCNSubsystemSignalingFlagUL")
			case nasMessage.DNSServerIPv6AddressRequestUL:
				smCtx.ProtocolConfigurationOptions.DNSIPv6Request = true
			case nasMessage.NotSupportedUL:
				logger.GsmLog.Infoln("Didn't Implement container type NotSupportedUL")
			case nasMessage.MSSupportOfNetworkRequestedBearerControlIndicatorUL:
				logger.GsmLog.Infoln("Didn't Implement container type MSSupportOfNetworkRequestedBearerControlIndicatorUL")
			case nasMessage.DSMIPv6HomeAgentAddressRequestUL:
				logger.GsmLog.Infoln("Didn't Implement container type DSMIPv6HomeAgentAddressRequestUL")
			case nasMessage.DSMIPv6HomeNetworkPrefixRequestUL:
				logger.GsmLog.Infoln("Didn't Implement container type DSMIPv6HomeNetworkPrefixRequestUL")
			case nasMessage.DSMIPv6IPv4HomeAgentAddressRequestUL:
				logger.GsmLog.Infoln("Didn't Implement container type DSMIPv6IPv4HomeAgentAddressRequestUL")
			case nasMessage.IPAddressAllocationViaNASSignallingUL:
				logger.GsmLog.Infoln("Didn't Implement container type IPAddressAllocationViaNASSignallingUL")
			case nasMessage.IPv4AddressAllocationViaDHCPv4UL:
				logger.GsmLog.Infoln("Didn't Implement container type IPv4AddressAllocationViaDHCPv4UL")
			case nasMessage.PCSCFIPv4AddressRequestUL:
				smCtx.ProtocolConfigurationOptions.PCSCFIPv4Request = true
			case nasMessage.DNSServerIPv4AddressRequestUL:
				smCtx.ProtocolConfigurationOptions.DNSIPv4Request = true
			case nasMessage.MSISDNRequestUL:
				logger.GsmLog.Infoln("Didn't Implement container type MSISDNRequestUL")
			case nasMessage.IFOMSupportRequestUL:
				logger.GsmLog.Infoln("Didn't Implement container type IFOMSupportRequestUL")
			case nasMessage.IPv4LinkMTURequestUL:
				smCtx.ProtocolConfigurationOptions.IPv4LinkMTURequest = true
			case nasMessage.MSSupportOfLocalAddressInTFTIndicatorUL:
				logger.GsmLog.Infoln("Didn't Implement container type MSSupportOfLocalAddressInTFTIndicatorUL")
			case nasMessage.PCSCFReSelectionSupportUL:
				logger.GsmLog.Infoln("Didn't Implement container type PCSCFReSelectionSupportUL")
			case nasMessage.NBIFOMRequestIndicatorUL:
				logger.GsmLog.Infoln("Didn't Implement container type NBIFOMRequestIndicatorUL")
			case nasMessage.NBIFOMModeUL:
				logger.GsmLog.Infoln("Didn't Implement container type NBIFOMModeUL")
			case nasMessage.NonIPLinkMTURequestUL:
				logger.GsmLog.Infoln("Didn't Implement container type NonIPLinkMTURequestUL")
			case nasMessage.APNRateControlSupportIndicatorUL:
				logger.GsmLog.Infoln("Didn't Implement container type APNRateControlSupportIndicatorUL")
			case nasMessage.UEStatus3GPPPSDataOffUL:
				logger.GsmLog.Infoln("Didn't Implement container type UEStatus3GPPPSDataOffUL")
			case nasMessage.ReliableDataServiceRequestIndicatorUL:
				logger.GsmLog.Infoln("Didn't Implement container type ReliableDataServiceRequestIndicatorUL")
			case nasMessage.AdditionalAPNRateControlForExceptionDataSupportIndicatorUL:
				logger.GsmLog.Infoln(
					"Didn't Implement container type AdditionalAPNRateControlForExceptionDataSupportIndicatorUL",
				)
			case nasMessage.PDUSessionIDUL:
				logger.GsmLog.Infoln("Didn't Implement container type PDUSessionIDUL")
			case nasMessage.EthernetFramePayloadMTURequestUL:
				logger.GsmLog.Infoln("Didn't Implement container type EthernetFramePayloadMTURequestUL")
			case nasMessage.UnstructuredLinkMTURequestUL:
				logger.GsmLog.Infoln("Didn't Implement container type UnstructuredLinkMTURequestUL")
			case nasMessage.I5GSMCauseValueUL:
				logger.GsmLog.Infoln("Didn't Implement container type 5GSMCauseValueUL")
			case nasMessage.QoSRulesWithTheLengthOfTwoOctetsSupportIndicatorUL:
				logger.GsmLog.Infoln("Didn't Implement container type QoSRulesWithTheLengthOfTwoOctetsSupportIndicatorUL")
			case nasMessage.QoSFlowDescriptionsWithTheLengthOfTwoOctetsSupportIndicatorUL:
				logger.GsmLog.Infoln(
					"Didn't Implement container type QoSFlowDescriptionsWithTheLengthOfTwoOctetsSupportIndicatorUL",
				)
			case nasMessage.LinkControlProtocolUL:
				logger.GsmLog.Infoln("Didn't Implement container type LinkControlProtocolUL")
			case nasMessage.PushAccessControlProtocolUL:
				logger.GsmLog.Infoln("Didn't Implement container type PushAccessControlProtocolUL")
			case nasMessage.ChallengeHandshakeAuthenticationProtocolUL:
				logger.GsmLog.Infoln("Didn't Implement container type ChallengeHandshakeAuthenticationProtocolUL")
			case nasMessage.InternetProtocolControlProtocolUL:
				logger.GsmLog.Infoln("Didn't Implement container type InternetProtocolControlProtocolUL")
			default:
				logger.GsmLog.Infof("Unknown Container ID [%d]", container.ProtocolOrContainerID)
			}
		}
	}
}

func HandlePDUSessionReleaseRequest(
	smCtx *smf_context.SMContext, req *nasMessage.PDUSessionReleaseRequest) {
	logger.GsmLog.Infof("Handle Pdu Session Release Request")

	// Retrieve PTI (Procedure transaction identity)
	smCtx.Pti = req.GetPTI()
}

func HandlePDUSessionModificationRequest(
	smCtx *smf_context.SMContext, req *nasMessage.PDUSessionModificationRequest) (*nas.Message, error) {
	logger.GsmLog.Infof("Handle Pdu Session Modification Request")

	// Retrieve PTI (Procedure transaction identity)
	smCtx.Pti = req.GetPTI()

	rsp := nas.NewMessage()
	rsp.GsmMessage = nas.NewGsmMessage()
	rsp.GsmHeader.SetMessageType(nas.MsgTypePDUSessionModificationCommand)
	rsp.GsmHeader.SetExtendedProtocolDiscriminator(nasMessage.Epd5GSSessionManagementMessage)
	rsp.PDUSessionModificationCommand = nasMessage.NewPDUSessionModificationCommand(0x00)
	pDUSessionModificationCommand := rsp.PDUSessionModificationCommand
	pDUSessionModificationCommand.SetExtendedProtocolDiscriminator(nasMessage.Epd5GSSessionManagementMessage)
	pDUSessionModificationCommand.SetPDUSessionID(uint8(smCtx.PDUSessionID))
	pDUSessionModificationCommand.SetPTI(smCtx.Pti)
	pDUSessionModificationCommand.SetMessageType(nas.MsgTypePDUSessionModificationCommand)

	reqQoSRules := nasType.QoSRules{}
	reqQoSFlowDescs := nasType.QoSFlowDescs{}

	if req.RequestedQosRules != nil {
		qosRuleBytes := req.GetQoSRules()
		if err := reqQoSRules.UnmarshalBinary(qosRuleBytes); err != nil {
			smCtx.Log.Warning("QoS rule parse failed:", err)
		}
	}

	if req.RequestedQosFlowDescriptions != nil {
		qosFlowDescsBytes := req.GetQoSFlowDescriptions()
		if err := reqQoSFlowDescs.UnmarshalBinary(qosFlowDescsBytes); err != nil {
			smCtx.Log.Warning("QoS flow descriptions parse failed:", err)
		}
	}

	var smPolicyDecision *models.SmPolicyDecision

	if smPolicyDecisionRsp, err := consumer.
		SendSMPolicyAssociationUpdateByUERequestModification(smCtx, reqQoSRules, reqQoSFlowDescs); err != nil {
	} else {
		smPolicyDecision = smPolicyDecisionRsp
	}

	authQoSRules := nasType.QoSRules{}
	authQoSFlowDesc := reqQoSFlowDescs

	for _, pcc := range smPolicyDecision.PccRules {
		rule := nasType.QoSRule{}
		rule.Operation = reqQoSRules[0].Operation
		rule.Precedence = uint8(pcc.Precedence)

		if ruleID, err := smCtx.QoSRuleIDGenerator.Allocate(); err != nil {
			return nil, err
		} else {
			rule.Identifier = uint8(ruleID)
			smCtx.PCCRuleIDToQoSRuleID[pcc.PccRuleId] = uint8(ruleID)
		}

		pfList := make(nasType.PacketFilterList, 0)
		for _, flowInfo := range pcc.FlowInfos {
			if pf, err := buildNASPacketFilterFromFlowInformation(&flowInfo); err != nil {
				smCtx.Log.Warning("Build packet filter fail:", err)
				continue
			} else {
				if packetFilterID, err := smCtx.PacketFilterIDGenerator.Allocate(); err != nil {
					return nil, err
				} else {
					pf.Identifier = uint8(packetFilterID)
					smCtx.PacketFilterIDToNASPFID[flowInfo.PackFiltId] = uint8(packetFilterID)
				}

				pfList = append(pfList, *pf)
			}
		}
		rule.PacketFilterList = pfList
		qosRef := pcc.RefQosData[0]
		qosDesc := smPolicyDecision.QosDecs[qosRef]
		rule.QFI = uint8(qosDesc.Var5qi)

		authQoSRules = append(authQoSRules, rule)
	}

	if len(authQoSRules) > 0 {
		if buf, err := authQoSRules.MarshalBinary(); err != nil {
			return nil, err
		} else {
			pDUSessionModificationCommand.AuthorizedQosRules = nasType.NewAuthorizedQosRules(0x7A)
			pDUSessionModificationCommand.AuthorizedQosRules.SetLen(uint16(len(buf)))
			pDUSessionModificationCommand.AuthorizedQosRules.SetQosRule(buf)
		}
	}

	if len(authQoSFlowDesc) > 0 {
		if buf, err := authQoSFlowDesc.MarshalBinary(); err != nil {
			return nil, err
		} else {
			pDUSessionModificationCommand.AuthorizedQosFlowDescriptions = nasType.NewAuthorizedQosFlowDescriptions(0x79)
			pDUSessionModificationCommand.AuthorizedQosFlowDescriptions.SetLen(uint16(len(buf)))
			pDUSessionModificationCommand.AuthorizedQosFlowDescriptions.SetQoSFlowDescriptions(buf)
		}
	}

	return rsp, nil
}

func buildNASPacketFilterFromFlowInformation(pfInfo *models.FlowInformation) (*nasType.PacketFilter, error) {
	nasPacketFilter := new(nasType.PacketFilter)

	// set flow direction
	switch pfInfo.FlowDirection {
	case models.FlowDirectionRm_BIDIRECTIONAL:
		nasPacketFilter.Direction = nasType.PacketFilterDirectionBidirectional
	case models.FlowDirectionRm_DOWNLINK:
		nasPacketFilter.Direction = nasType.PacketFilterDirectionDownlink
	case models.FlowDirectionRm_UPLINK:
		nasPacketFilter.Direction = nasType.PacketFilterDirectionUplink
	}

	pfComponents := make(nasType.PacketFilterComponentList, 0)

	if pfInfo.FlowLabel != "" {
		if label, err := strconv.ParseInt(pfInfo.FlowLabel, 16, 32); err != nil {
			return nil, fmt.Errorf("parse flow label fail: %s", err)
		} else {
			pfComponents = append(pfComponents, &nasType.PacketFilterFlowLabel{
				Label: uint32(label),
			})
		}
	}

	if pfInfo.Spi != "" {
		if spi, err := strconv.ParseInt(pfInfo.Spi, 16, 32); err != nil {
			return nil, fmt.Errorf("parse security parameter index fail: %s", err)
		} else {
			pfComponents = append(pfComponents, &nasType.PacketFilterSecurityParameterIndex{
				Index: uint32(spi),
			})
		}
	}

	if pfInfo.TosTrafficClass != "" {
		if tos, err := strconv.ParseInt(pfInfo.TosTrafficClass, 16, 32); err != nil {
			return nil, fmt.Errorf("parse security parameter index fail: %s", err)
		} else {
			pfComponents = append(pfComponents, &nasType.PacketFilterServiceClass{
				Class: uint8(tos >> 8),
				Mask:  uint8(tos & 0x00FF),
			})
		}
	}

	if pfInfo.FlowDescription != "" {
		ipFilter := flowdesc.NewIPFilterRule()
		if err := flowdesc.Decode(pfInfo.FlowDescription, ipFilter); err != nil {
			return nil, fmt.Errorf("parse packet filter content fail: %s", err)
		}
		pfComponents = append(pfComponents, buildPacketFilterComponentsFromIPFilterRule(ipFilter)...)
	}

	if len(pfComponents) == 0 {
		pfComponents = append(pfComponents, &nasType.PacketFilterMatchAll{})
	}

	nasPacketFilter.Components = pfComponents

	return nasPacketFilter, nil
}

func buildPacketFilterComponentsFromIPFilterRule(filterRule *flowdesc.IPFilterRule) nasType.PacketFilterComponentList {
	pfComponents := make(nasType.PacketFilterComponentList, 0)

	srcIP := filterRule.GetSourceIP()
	srcPorts := filterRule.GetSourcePorts()
	dstIP := filterRule.GetDestinationIP()
	dstPorts := filterRule.GetDestinationPorts()
	proto := filterRule.GetProtocol()

	if srcIP != "any" {
		_, ipNet, err := net.ParseCIDR(srcIP)
		if err != nil {
			return nil
		}
		pfComponents = append(pfComponents, &nasType.PacketFilterIPv4RemoteAddress{
			Address: ipNet.IP.To4(),
			Mask:    ipNet.Mask,
		})
	}

	if dstIP != "any" {
		_, ipNet, err := net.ParseCIDR(dstIP)
		if err != nil {
			return nil
		}
		pfComponents = append(pfComponents, &nasType.PacketFilterIPv4LocalAddress{
			Address: ipNet.IP.To4(),
			Mask:    ipNet.Mask,
		})
	}

	if dstPorts != "" {
		ports := strings.Split(dstPorts, "-")
		if len(ports) == 2 {
			var low, high uint16
			if ret, err := strconv.Atoi(ports[0]); err != nil {
				return nil
			} else {
				low = uint16(ret)
			}

			if ret, err := strconv.Atoi(ports[1]); err != nil {
				return nil
			} else {
				high = uint16(ret)
			}
			pfComponents = append(pfComponents, &nasType.PacketFilterLocalPortRange{
				LowLimit:  low,
				HighLimit: high,
			})
		} else {
			if ret, err := strconv.Atoi(ports[0]); err != nil {
				return nil
			} else {
				pfComponents = append(pfComponents, &nasType.PacketFilterSingleLocalPort{
					Value: uint16(ret),
				})
			}
		}
	}

	if srcPorts != "" {
		ports := strings.Split(srcPorts, "-")
		if len(ports) == 2 {
			var low, high uint16
			if ret, err := strconv.Atoi(ports[0]); err != nil {
				return nil
			} else {
				low = uint16(ret)
			}

			if ret, err := strconv.Atoi(ports[1]); err != nil {
				return nil
			} else {
				high = uint16(ret)
			}
			pfComponents = append(pfComponents, &nasType.PacketFilterRemotePortRange{
				LowLimit:  low,
				HighLimit: high,
			})
		} else {
			if ret, err := strconv.Atoi(ports[0]); err != nil {
				return nil
			} else {
				pfComponents = append(pfComponents, &nasType.PacketFilterSingleRemotePort{
					Value: uint16(ret),
				})
			}
		}
	}

	if proto != flowdesc.ProtocolNumberAny {
		pfComponents = append(pfComponents, &nasType.PacketFilterProtocolIdentifier{
			Value: proto,
		})
	}

	return pfComponents
}
