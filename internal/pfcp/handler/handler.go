package handler

import (
	"context"
	"fmt"

	"bitbucket.org/free5gc-team/openapi/models"
	"bitbucket.org/free5gc-team/pfcp"
	"bitbucket.org/free5gc-team/pfcp/pfcpType"
	"bitbucket.org/free5gc-team/pfcp/pfcpUdp"
	smf_context "bitbucket.org/free5gc-team/smf/internal/context"
	"bitbucket.org/free5gc-team/smf/internal/logger"
	pfcp_message "bitbucket.org/free5gc-team/smf/internal/pfcp/message"
	"bitbucket.org/free5gc-team/smf/internal/sbi/producer"
)

func HandlePfcpHeartbeatRequest(msg *pfcpUdp.Message) {
	h := msg.PfcpMessage.Header
	pfcp_message.SendHeartbeatResponse(msg.RemoteAddr, h.SequenceNumber)
}

func HandlePfcpPfdManagementRequest(msg *pfcpUdp.Message) {
	logger.PfcpLog.Warnf("PFCP PFD Management Request handling is not implemented")
}

func HandlePfcpAssociationSetupRequest(msg *pfcpUdp.Message) {
	req := msg.PfcpMessage.Body.(pfcp.PFCPAssociationSetupRequest)

	nodeID := req.NodeID
	if nodeID == nil {
		logger.PfcpLog.Errorln("pfcp association needs NodeID")
		return
	}
	logger.PfcpLog.Infof("Handle PFCP Association Setup Request with NodeID[%s]", nodeID.ResolveNodeIdToIp().String())

	upf := smf_context.RetrieveUPFNodeByNodeID(*nodeID)
	if upf == nil {
		logger.PfcpLog.Errorf("can't find UPF[%s]", nodeID.ResolveNodeIdToIp().String())
		return
	}

	upf.UPIPInfo = *req.UserPlaneIPResourceInformation

	// Response with PFCP Association Setup Response
	cause := pfcpType.Cause{
		CauseValue: pfcpType.CauseRequestAccepted,
	}
	pfcp_message.SendPfcpAssociationSetupResponse(msg.RemoteAddr, cause)
}

func nodeIDtoString(nodeID *pfcpType.NodeID) string {
	switch nodeID.NodeIdType {
	case pfcpType.NodeIdTypeIpv4Address:
		fallthrough
	case pfcpType.NodeIdTypeIpv6Address:
		return nodeID.IP.String()
	case pfcpType.NodeIdTypeFqdn:
		return nodeID.FQDN
	default:
		return ""
	}
}

func HandlePfcpAssociationSetupResponse(msg *pfcpUdp.Message) {
	rsp := msg.PfcpMessage.Body.(pfcp.PFCPAssociationSetupResponse)
	logger.PfcpLog.Infoln("In HandlePfcpAssociationSetupResponse")

	nodeID := rsp.NodeID
	if rsp.Cause.CauseValue == pfcpType.CauseRequestAccepted {
		if nodeID == nil {
			logger.PfcpLog.Errorln("pfcp association needs NodeID")
			return
		}
		logger.PfcpLog.Infof("Handle PFCP Association Setup Response with NodeID[%s]", nodeIDtoString(nodeID))

		upf := smf_context.RetrieveUPFNodeByNodeID(*nodeID)
		if upf == nil {
			logger.PfcpLog.Errorf("can't find UPF[%s]", nodeIDtoString(nodeID))
			return
		}

		upf.UPFStatus = smf_context.AssociatedSetUpSuccess
	}
}

func HandlePfcpAssociationUpdateRequest(msg *pfcpUdp.Message) {
	logger.PfcpLog.Warnf("PFCP Association Update Request handling is not implemented")
}

func HandlePfcpAssociationReleaseRequest(msg *pfcpUdp.Message) {
	pfcpMsg := msg.PfcpMessage.Body.(pfcp.PFCPAssociationReleaseRequest)

	var cause pfcpType.Cause
	upf := smf_context.RetrieveUPFNodeByNodeID(*pfcpMsg.NodeID)

	if upf != nil {
		smf_context.RemoveUPFNodeByNodeID(*pfcpMsg.NodeID)
		cause.CauseValue = pfcpType.CauseRequestAccepted
	} else {
		cause.CauseValue = pfcpType.CauseNoEstablishedPfcpAssociation
	}

	pfcp_message.SendPfcpAssociationReleaseResponse(msg.RemoteAddr, cause)
}

func HandlePfcpNodeReportRequest(msg *pfcpUdp.Message) {
	logger.PfcpLog.Warnf("PFCP Node Report Request handling is not implemented")
}

func HandlePfcpSessionSetDeletionRequest(msg *pfcpUdp.Message) {
	logger.PfcpLog.Warnf("PFCP Session Set Deletion Request handling is not implemented")
}

func HandlePfcpSessionSetDeletionResponse(msg *pfcpUdp.Message) {
	logger.PfcpLog.Warnf("PFCP Session Set Deletion Response handling is not implemented")
}

func HandlePfcpSessionEstablishmentResponse(msg *pfcpUdp.Message) {
	rsp := msg.PfcpMessage.Body.(pfcp.PFCPSessionEstablishmentResponse)
	logger.PfcpLog.Infoln("In HandlePfcpSessionEstablishmentResponse")

	SEID := msg.PfcpMessage.Header.SEID
	smContext := smf_context.GetSMContextBySEID(SEID)

	if smContext == nil {
		logger.PfcpLog.Errorf("PFCP Session SEID[%d] not found", SEID)
		return
	}

	if rsp.UPFSEID != nil {
		NodeIDtoIP := rsp.NodeID.ResolveNodeIdToIp().String()
		pfcpSessionCtx := smContext.PFCPContext[NodeIDtoIP]
		pfcpSessionCtx.RemoteSEID = rsp.UPFSEID.Seid
	}

	ANUPF := smContext.Tunnel.DataPathPool.GetDefaultPath().FirstDPNode

	// If this message is for the anchor UPF, trigger n1n2 transfer to amf and unlock the context lock
	if rsp.Cause.CauseValue == pfcpType.CauseRequestAccepted &&
		ANUPF.UPF.NodeID.ResolveNodeIdToIp().Equal(rsp.NodeID.ResolveNodeIdToIp()) {
		defer smContext.SMLock.Unlock()
		smContext.SMLockTimer.Stop()

		n1n2Request := models.N1N2MessageTransferRequest{}

		if smNasBuf, err := smf_context.BuildGSMPDUSessionEstablishmentAccept(smContext); err != nil {
			logger.PduSessLog.Errorf("Build GSM PDUSessionEstablishmentAccept failed: %s", err)
		} else {
			n1n2Request.BinaryDataN1Message = smNasBuf
		}
		if n2Pdu, err := smf_context.BuildPDUSessionResourceSetupRequestTransfer(smContext); err != nil {
			logger.PduSessLog.Errorf("Build PDUSessionResourceSetupRequestTransfer failed: %s", err)
		} else {
			n1n2Request.BinaryDataN2Information = n2Pdu
		}

		n1n2Request.JsonData = &models.N1N2MessageTransferReqData{
			PduSessionId: smContext.PDUSessionID,
			N1MessageContainer: &models.N1MessageContainer{
				N1MessageClass:   "SM",
				N1MessageContent: &models.RefToBinaryData{ContentId: "GSM_NAS"},
			},
			N2InfoContainer: &models.N2InfoContainer{
				N2InformationClass: models.N2InformationClass_SM,
				SmInfo: &models.N2SmInformation{
					PduSessionId: smContext.PDUSessionID,
					N2InfoContent: &models.N2InfoContent{
						NgapIeType: models.NgapIeType_PDU_RES_SETUP_REQ,
						NgapData: &models.RefToBinaryData{
							ContentId: "N2SmInformation",
						},
					},
					SNssai: smContext.SNssai,
				},
			},
		}

		rspData, _, err := smContext.
			CommunicationClient.
			N1N2MessageCollectionDocumentApi.
			N1N2MessageTransfer(context.Background(), smContext.Supi, n1n2Request)
		smContext.SetState(smf_context.Active)
		if err != nil {
			logger.PfcpLog.Warnf("Send N1N2Transfer failed: %s", err)
		}
		if rspData.Cause == models.N1N2MessageTransferCause_N1_MSG_NOT_TRANSFERRED {
			logger.PfcpLog.Warnf("%v", rspData.Cause)
		}
	}

	if smf_context.SMF_Self().ULCLSupport && smContext.BPManager != nil {
		if smContext.BPManager.BPStatus == smf_context.AddingPSA {
			logger.PfcpLog.Infoln("Keep Adding PSAndULCL")
			producer.AddPDUSessionAnchorAndULCL(smContext)
			smContext.BPManager.BPStatus = smf_context.AddingPSA
		}
	}
}

func HandlePfcpSessionReportRequest(msg *pfcpUdp.Message) {
	req := msg.PfcpMessage.Body.(pfcp.PFCPSessionReportRequest)

	SEID := msg.PfcpMessage.Header.SEID
	smContext := smf_context.GetSMContextBySEID(SEID)
	seqFromUPF := msg.PfcpMessage.Header.SequenceNumber

	var cause pfcpType.Cause

	if smContext == nil {
		logger.PfcpLog.Errorf("PFCP Session SEID[%d] not found", SEID)
		cause.CauseValue = pfcpType.CauseSessionContextNotFound
		pfcp_message.SendPfcpSessionReportResponse(msg.RemoteAddr, cause, seqFromUPF, 0)
		return
	}

	smContext.SMLock.Lock()
	defer smContext.SMLock.Unlock()

	if smContext.UpCnxState == models.UpCnxState_DEACTIVATED {
		if req.ReportType.Dldr {
			downlinkDataReport := req.DownlinkDataReport

			if downlinkDataReport.DownlinkDataServiceInformation != nil {
				logger.PfcpLog.Warnf("PFCP Session Report Request DownlinkDataServiceInformation handling is not implemented")
			}

			n1n2Request := models.N1N2MessageTransferRequest{}

			// TS 23.502 4.2.3.3 3a. Send Namf_Communication_N1N2MessageTransfer Request, SMF->AMF
			if n2SmBuf, err := smf_context.BuildPDUSessionResourceSetupRequestTransfer(smContext); err != nil {
				logger.PduSessLog.Errorln("Build PDUSessionResourceSetupRequestTransfer failed:", err)
			} else {
				n1n2Request.BinaryDataN2Information = n2SmBuf
			}

			n1n2Request.JsonData = &models.N1N2MessageTransferReqData{
				PduSessionId: smContext.PDUSessionID,
				// Temporarily assign SMF itself, TODO: TS 23.502 4.2.3.3 5. Namf_Communication_N1N2TransferFailureNotification
				N1n2FailureTxfNotifURI: fmt.Sprintf("%s://%s:%d",
					smf_context.SMF_Self().URIScheme,
					smf_context.SMF_Self().RegisterIPv4,
					smf_context.SMF_Self().SBIPort),
				N2InfoContainer: &models.N2InfoContainer{
					N2InformationClass: models.N2InformationClass_SM,
					SmInfo: &models.N2SmInformation{
						PduSessionId: smContext.PDUSessionID,
						N2InfoContent: &models.N2InfoContent{
							NgapIeType: models.NgapIeType_PDU_RES_SETUP_REQ,
							NgapData: &models.RefToBinaryData{
								ContentId: "N2SmInformation",
							},
						},
						SNssai: smContext.SNssai,
					},
				},
			}

			rspData, _, err := smContext.CommunicationClient.
				N1N2MessageCollectionDocumentApi.
				N1N2MessageTransfer(context.Background(), smContext.Supi, n1n2Request)
			if err != nil {
				logger.PfcpLog.Warnf("Send N1N2Transfer failed: %s", err)
			}
			if rspData.Cause == models.N1N2MessageTransferCause_ATTEMPTING_TO_REACH_UE {
				logger.PfcpLog.Infof("Receive %v, AMF is able to page the UE", rspData.Cause)
			}
			if rspData.Cause == models.N1N2MessageTransferCause_UE_NOT_RESPONDING {
				logger.PfcpLog.Warnf("%v", rspData.Cause)
				// TODO: TS 23.502 4.2.3.3 3c. Failure indication
			}
		}
	}

	// TS 23.502 4.2.3.3 2b. Send Data Notification Ack, SMF->UPF
	cause.CauseValue = pfcpType.CauseRequestAccepted
	// TODO fix: SEID should be the value sent by UPF but now the SEID value is from sm context
	pfcp_message.SendPfcpSessionReportResponse(msg.RemoteAddr, cause, seqFromUPF, SEID)
}
