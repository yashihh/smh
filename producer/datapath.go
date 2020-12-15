package producer

import (
	smf_context "bitbucket.org/free5gc-team/smf/context"
	"bitbucket.org/free5gc-team/smf/logger"
	pfcp_message "bitbucket.org/free5gc-team/smf/pfcp/message"
)

func SendPFCPRule(smContext *smf_context.SMContext, dataPath *smf_context.DataPath) {

	logger.PduSessLog.Infoln("Send PFCP Rule")
	logger.PduSessLog.Infoln("DataPath: ", dataPath)
	for curDataPathNode := dataPath.FirstDPNode; curDataPathNode != nil; curDataPathNode = curDataPathNode.Next() {
		pdrList := make([]*smf_context.PDR, 0, 2)
		farList := make([]*smf_context.FAR, 0, 2)
		qerList := make([]*smf_context.QER, 0, 2)

		sessionContext, exist := smContext.PFCPContext[curDataPathNode.GetNodeIP()]
		if !exist || sessionContext.RemoteSEID == 0 {
			if curDataPathNode.UpLinkTunnel != nil && curDataPathNode.UpLinkTunnel.PDR != nil {
				pdrList = append(pdrList, curDataPathNode.UpLinkTunnel.PDR)
				farList = append(farList, curDataPathNode.UpLinkTunnel.PDR.FAR)
				if curDataPathNode.UpLinkTunnel.PDR.QER != nil {
					qerList = append(qerList, curDataPathNode.UpLinkTunnel.PDR.QER)
				}
			}
			if curDataPathNode.DownLinkTunnel != nil && curDataPathNode.DownLinkTunnel.PDR != nil {
				pdrList = append(pdrList, curDataPathNode.DownLinkTunnel.PDR)
				farList = append(farList, curDataPathNode.DownLinkTunnel.PDR.FAR)
				// skip send QER because uplink and downlink shared one QER
			}

			pfcp_message.SendPfcpSessionEstablishmentRequest(curDataPathNode.UPF.NodeID, smContext, pdrList, farList, nil, qerList)
		} else {
			if curDataPathNode.UpLinkTunnel != nil && curDataPathNode.UpLinkTunnel.PDR != nil {
				pdrList = append(pdrList, curDataPathNode.UpLinkTunnel.PDR)
				farList = append(farList, curDataPathNode.UpLinkTunnel.PDR.FAR)
				if curDataPathNode.DownLinkTunnel.PDR.QER != nil {
					qerList = append(qerList, curDataPathNode.DownLinkTunnel.PDR.QER)
				}
			}
			if curDataPathNode.DownLinkTunnel != nil && curDataPathNode.DownLinkTunnel.PDR != nil {
				pdrList = append(pdrList, curDataPathNode.DownLinkTunnel.PDR)
				farList = append(farList, curDataPathNode.DownLinkTunnel.PDR.FAR)
				if curDataPathNode.DownLinkTunnel.PDR.QER != nil {
					qerList = append(qerList, curDataPathNode.DownLinkTunnel.PDR.QER)
				}
			}

			pfcp_message.SendPfcpSessionModificationRequest(curDataPathNode.UPF.NodeID, smContext, pdrList, farList, nil, qerList)
		}

	}
}

func createPccRuleDataPath(smContext *smf_context.SMContext,
	pccRule *smf_context.PCCRule,
	tcData *smf_context.TrafficControlData) {
	var targetDNAI string
	if tcData != nil && len(tcData.RouteToLocs) > 0 {
		targetDNAI = tcData.RouteToLocs[0].Dnai
	}
	upfSelectionParams := &smf_context.UPFSelectionParams{
		Dnn: smContext.Dnn,
		SNssai: &smf_context.SNssai{
			Sst: smContext.Snssai.Sst,
			Sd:  smContext.Snssai.Sd,
		},
		Dnai: targetDNAI,
	}
	createdUpPath := smf_context.GetUserPlaneInformation().GetDefaultUserPlanePathByDNN(upfSelectionParams)
	createdDataPath := smf_context.GenerateDataPath(createdUpPath, smContext)
	if createdDataPath != nil {
		createdDataPath.ActivateTunnelAndPDR(smContext)
		smContext.Tunnel.AddDataPath(createdDataPath)
	}

	pccRule.Datapath = createdDataPath
}

func removeDataPath(smContext *smf_context.SMContext, datapath *smf_context.DataPath) {
	for curDPNode := datapath.FirstDPNode; curDPNode != nil; curDPNode = curDPNode.Next() {
		if curDPNode.DownLinkTunnel != nil && curDPNode.DownLinkTunnel.PDR != nil {
			curDPNode.DownLinkTunnel.PDR.State = smf_context.RULE_REMOVE
			curDPNode.DownLinkTunnel.PDR.FAR.State = smf_context.RULE_REMOVE
		}
		if curDPNode.UpLinkTunnel != nil && curDPNode.UpLinkTunnel.PDR != nil {
			curDPNode.UpLinkTunnel.PDR.State = smf_context.RULE_REMOVE
			curDPNode.UpLinkTunnel.PDR.FAR.State = smf_context.RULE_REMOVE
		}
	}
}

// UpdateDataPathToUPF update the datapath of the UPF
func UpdateDataPathToUPF(smContext *smf_context.SMContext, oldDataPath, updateDataPath *smf_context.DataPath) {
	if oldDataPath == nil {
		SendPFCPRule(smContext, updateDataPath)
		return
	} else {
		removeDataPath(smContext, oldDataPath)
		SendPFCPRule(smContext, updateDataPath)
	}
}
