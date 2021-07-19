package message

import (
	"net"

	"bitbucket.org/free5gc-team/pfcp"
	"bitbucket.org/free5gc-team/pfcp/pfcpType"
	"bitbucket.org/free5gc-team/pfcp/pfcpUdp"
	"bitbucket.org/free5gc-team/smf/internal/context"
	"bitbucket.org/free5gc-team/smf/internal/logger"
	"bitbucket.org/free5gc-team/smf/internal/pfcp/udp"
)

var seq uint32

func getSeqNumber() uint32 {
	seq++
	return seq
}

func SendPfcpAssociationSetupRequest(upN4Addr string) {
	pfcpMsg, err := BuildPfcpAssociationSetupRequest()
	if err != nil {
		logger.PfcpLog.Errorf("Build PFCP Association Setup Request failed: %v", err)
		return
	}

	message := pfcp.Message{
		Header: pfcp.Header{
			Version:        pfcp.PfcpVersion,
			MP:             0,
			S:              pfcp.SEID_NOT_PRESENT,
			MessageType:    pfcp.PFCP_ASSOCIATION_SETUP_REQUEST,
			SequenceNumber: getSeqNumber(),
		},
		Body: pfcpMsg,
	}

	addr := &net.UDPAddr{
		IP:   context.ResolveIP(upN4Addr),
		Port: pfcpUdp.PFCP_PORT,
	}

	udp.SendPfcp(message, addr)
}

func SendPfcpAssociationSetupResponse(addr *net.UDPAddr, cause pfcpType.Cause) {
	pfcpMsg, err := BuildPfcpAssociationSetupResponse(cause)
	if err != nil {
		logger.PfcpLog.Errorf("Build PFCP Association Setup Response failed: %v", err)
		return
	}

	message := pfcp.Message{
		Header: pfcp.Header{
			Version:        pfcp.PfcpVersion,
			MP:             0,
			S:              pfcp.SEID_NOT_PRESENT,
			MessageType:    pfcp.PFCP_ASSOCIATION_SETUP_RESPONSE,
			SequenceNumber: 1,
		},
		Body: pfcpMsg,
	}

	udp.SendPfcp(message, addr)
}

func SendPfcpAssociationReleaseRequest(upN4Addr string) {
	pfcpMsg, err := BuildPfcpAssociationReleaseRequest()
	if err != nil {
		logger.PfcpLog.Errorf("Build PFCP Association Release Request failed: %v", err)
		return
	}

	message := pfcp.Message{
		Header: pfcp.Header{
			Version:        pfcp.PfcpVersion,
			MP:             0,
			S:              pfcp.SEID_NOT_PRESENT,
			MessageType:    pfcp.PFCP_ASSOCIATION_RELEASE_REQUEST,
			SequenceNumber: 1,
		},
		Body: pfcpMsg,
	}

	addr := &net.UDPAddr{
		IP:   context.ResolveIP(upN4Addr),
		Port: pfcpUdp.PFCP_PORT,
	}

	udp.SendPfcp(message, addr)
}

func SendPfcpAssociationReleaseResponse(addr *net.UDPAddr, cause pfcpType.Cause) {
	pfcpMsg, err := BuildPfcpAssociationReleaseResponse(cause)
	if err != nil {
		logger.PfcpLog.Errorf("Build PFCP Association Release Response failed: %v", err)
		return
	}

	message := pfcp.Message{
		Header: pfcp.Header{
			Version:        pfcp.PfcpVersion,
			MP:             0,
			S:              pfcp.SEID_NOT_PRESENT,
			MessageType:    pfcp.PFCP_ASSOCIATION_RELEASE_RESPONSE,
			SequenceNumber: 1,
		},
		Body: pfcpMsg,
	}

	udp.SendPfcp(message, addr)
}

func SendPfcpSessionEstablishmentRequest(
	upNodeID pfcpType.NodeID,
	upN4Addr string,
	ctx *context.SMContext,
	pdrList []*context.PDR, farList []*context.FAR, barList []*context.BAR, qerList []*context.QER) {
	pfcpMsg, err := BuildPfcpSessionEstablishmentRequest(upNodeID, upN4Addr, ctx, pdrList, farList, barList, qerList)
	if err != nil {
		logger.PfcpLog.Errorf("Build PFCP Session Establishment Request failed: %v", err)
		return
	}

	message := pfcp.Message{
		Header: pfcp.Header{
			Version:         pfcp.PfcpVersion,
			MP:              1,
			S:               pfcp.SEID_PRESENT,
			MessageType:     pfcp.PFCP_SESSION_ESTABLISHMENT_REQUEST,
			SEID:            0,
			SequenceNumber:  getSeqNumber(),
			MessagePriority: 0,
		},
		Body: pfcpMsg,
	}

	upaddr := &net.UDPAddr{
		IP:   context.ResolveIP(upN4Addr),
		Port: pfcpUdp.PFCP_PORT,
	}
	logger.PduSessLog.Traceln("[SMF] Send SendPfcpSessionEstablishmentRequest")
	logger.PduSessLog.Traceln("Send to addr ", upaddr.String())

	udp.SendPfcp(message, upaddr)
}

func SendPfcpSessionModificationRequest(upNodeID pfcpType.NodeID,
	upN4Addr string,
	ctx *context.SMContext,
	pdrList []*context.PDR, farList []*context.FAR, barList []*context.BAR, qerList []*context.QER) (seqNum uint32) {
	pfcpMsg, err := BuildPfcpSessionModificationRequest(upNodeID, upN4Addr, ctx, pdrList, farList, barList, qerList)
	if err != nil {
		logger.PfcpLog.Errorf("Build PFCP Session Modification Request failed: %v", err)
		return
	}

	seqNum = getSeqNumber()
	nodeIDtoIP := upNodeID.ResolveNodeIdToIp().String()
	remoteSEID := ctx.PFCPContext[nodeIDtoIP].RemoteSEID
	message := pfcp.Message{
		Header: pfcp.Header{
			Version:         pfcp.PfcpVersion,
			MP:              1,
			S:               pfcp.SEID_PRESENT,
			MessageType:     pfcp.PFCP_SESSION_MODIFICATION_REQUEST,
			SEID:            remoteSEID,
			SequenceNumber:  seqNum,
			MessagePriority: 12,
		},
		Body: pfcpMsg,
	}

	upaddr := &net.UDPAddr{
		IP:   context.ResolveIP(upN4Addr),
		Port: pfcpUdp.PFCP_PORT,
	}

	udp.SendPfcp(message, upaddr)
	return seqNum
}

func SendPfcpSessionDeletionRequest(upNodeID pfcpType.NodeID, upN4Addr string, ctx *context.SMContext) (seqNum uint32) {
	pfcpMsg, err := BuildPfcpSessionDeletionRequest()
	if err != nil {
		logger.PfcpLog.Errorf("Build PFCP Session Deletion Request failed: %v", err)
		return
	}
	seqNum = getSeqNumber()
	nodeIDtoIP := upNodeID.ResolveNodeIdToIp().String()
	remoteSEID := ctx.PFCPContext[nodeIDtoIP].RemoteSEID
	message := pfcp.Message{
		Header: pfcp.Header{
			Version:         pfcp.PfcpVersion,
			MP:              1,
			S:               pfcp.SEID_PRESENT,
			MessageType:     pfcp.PFCP_SESSION_DELETION_REQUEST,
			SEID:            remoteSEID,
			SequenceNumber:  seqNum,
			MessagePriority: 12,
		},
		Body: pfcpMsg,
	}

	upaddr := &net.UDPAddr{
		IP:   context.ResolveIP(upN4Addr),
		Port: pfcpUdp.PFCP_PORT,
	}

	udp.SendPfcp(message, upaddr)

	return seqNum
}

func SendPfcpSessionReportResponse(addr *net.UDPAddr, cause pfcpType.Cause, seqFromUPF uint32, SEID uint64) {
	pfcpMsg, err := BuildPfcpSessionReportResponse(cause)
	if err != nil {
		logger.PfcpLog.Errorf("Build PFCP Session Report Response failed: %v", err)
		return
	}

	message := pfcp.Message{
		Header: pfcp.Header{
			Version:        pfcp.PfcpVersion,
			MP:             0,
			S:              pfcp.SEID_PRESENT,
			MessageType:    pfcp.PFCP_SESSION_REPORT_RESPONSE,
			SequenceNumber: seqFromUPF,
			SEID:           SEID,
		},
		Body: pfcpMsg,
	}

	udp.SendPfcp(message, addr)
}

func SendHeartbeatResponse(addr *net.UDPAddr, seq uint32) {
	pfcpMsg := pfcp.HeartbeatResponse{
		RecoveryTimeStamp: &pfcpType.RecoveryTimeStamp{
			RecoveryTimeStamp: udp.ServerStartTime,
		},
	}

	message := pfcp.Message{
		Header: pfcp.Header{
			Version:        pfcp.PfcpVersion,
			MP:             0,
			S:              pfcp.SEID_NOT_PRESENT,
			MessageType:    pfcp.PFCP_HEARTBEAT_RESPONSE,
			SequenceNumber: seq,
		},
		Body: pfcpMsg,
	}

	udp.SendPfcp(message, addr)
}
