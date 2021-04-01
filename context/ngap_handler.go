package context

import (
	"encoding/binary"
	"errors"

	"bitbucket.org/free5gc-team/aper"
	"bitbucket.org/free5gc-team/ngap/ngapType"
	"bitbucket.org/free5gc-team/openapi/models"
	"bitbucket.org/free5gc-team/pfcp/pfcpType"
	"bitbucket.org/free5gc-team/smf/logger"
)

func HandlePDUSessionResourceSetupResponseTransfer(b []byte, ctx *SMContext) (err error) {
	resourceSetupResponseTransfer := ngapType.PDUSessionResourceSetupResponseTransfer{}

	err = aper.UnmarshalWithParams(b, &resourceSetupResponseTransfer, "valueExt")

	if err != nil {
		return err
	}

	QosFlowPerTNLInformation := resourceSetupResponseTransfer.DLQosFlowPerTNLInformation

	if QosFlowPerTNLInformation.UPTransportLayerInformation.Present !=
		ngapType.UPTransportLayerInformationPresentGTPTunnel {
		return errors.New("resourceSetupResponseTransfer.QosFlowPerTNLInformation.UPTransportLayerInformation.Present")
	}

	GTPTunnel := QosFlowPerTNLInformation.UPTransportLayerInformation.GTPTunnel

	ctx.Tunnel.UpdateANInformation(
		GTPTunnel.TransportLayerAddress.Value.Bytes,
		binary.BigEndian.Uint32(GTPTunnel.GTPTEID.Value))

	ctx.UpCnxState = models.UpCnxState_ACTIVATED
	return nil
}

func HandlePDUSessionResourceSetupUnsuccessfulTransfer(b []byte, ctx *SMContext) (err error) {
	resourceSetupUnsuccessfulTransfer := ngapType.PDUSessionResourceSetupUnsuccessfulTransfer{}

	err = aper.UnmarshalWithParams(b, &resourceSetupUnsuccessfulTransfer, "valueExt")

	if err != nil {
		return err
	}

	switch resourceSetupUnsuccessfulTransfer.Cause.Present {
	case ngapType.CausePresentRadioNetwork:
		logger.PduSessLog.Warnf("PDU Session Resource Setup Unsuccessful by RadioNetwork[%d]",
			resourceSetupUnsuccessfulTransfer.Cause.RadioNetwork.Value)
	case ngapType.CausePresentTransport:
		logger.PduSessLog.Warnf("PDU Session Resource Setup Unsuccessful by Transport[%d]",
			resourceSetupUnsuccessfulTransfer.Cause.Transport.Value)
	case ngapType.CausePresentNas:
		logger.PduSessLog.Warnf("PDU Session Resource Setup Unsuccessful by NAS[%d]",
			resourceSetupUnsuccessfulTransfer.Cause.Nas.Value)
	case ngapType.CausePresentProtocol:
		logger.PduSessLog.Warnf("PDU Session Resource Setup Unsuccessful by Protocol[%d]",
			resourceSetupUnsuccessfulTransfer.Cause.Protocol.Value)
	case ngapType.CausePresentMisc:
		logger.PduSessLog.Warnf("PDU Session Resource Setup Unsuccessful by Protocol[%d]",
			resourceSetupUnsuccessfulTransfer.Cause.Misc.Value)
	case ngapType.CausePresentChoiceExtensions:
		logger.PduSessLog.Warnf("PDU Session Resource Setup Unsuccessful by Protocol[%v]",
			resourceSetupUnsuccessfulTransfer.Cause.ChoiceExtensions)
	}

	ctx.UpCnxState = models.UpCnxState_ACTIVATING

	return nil
}

func HandlePathSwitchRequestTransfer(b []byte, ctx *SMContext) error {
	pathSwitchRequestTransfer := ngapType.PathSwitchRequestTransfer{}

	if err := aper.UnmarshalWithParams(b, &pathSwitchRequestTransfer, "valueExt"); err != nil {
		return err
	}

	if pathSwitchRequestTransfer.DLNGUUPTNLInformation.Present != ngapType.UPTransportLayerInformationPresentGTPTunnel {
		return errors.New("pathSwitchRequestTransfer.DLNGUUPTNLInformation.Present")
	}

	GTPTunnel := pathSwitchRequestTransfer.DLNGUUPTNLInformation.GTPTunnel

	ctx.Tunnel.UpdateANInformation(
		GTPTunnel.TransportLayerAddress.Value.Bytes,
		binary.BigEndian.Uint32(GTPTunnel.GTPTEID.Value))

	return nil
}

func HandlePathSwitchRequestSetupFailedTransfer(b []byte, ctx *SMContext) (err error) {
	pathSwitchRequestSetupFailedTransfer := ngapType.PathSwitchRequestSetupFailedTransfer{}

	err = aper.UnmarshalWithParams(b, &pathSwitchRequestSetupFailedTransfer, "valueExt")

	if err != nil {
		return err
	}

	// TODO: finish handler
	return nil
}

func HandleHandoverRequiredTransfer(b []byte, ctx *SMContext) (err error) {
	handoverRequiredTransfer := ngapType.HandoverRequiredTransfer{}

	err = aper.UnmarshalWithParams(b, &handoverRequiredTransfer, "valueExt")

	directForwardingPath := handoverRequiredTransfer.DirectForwardingPathAvailability
	if directForwardingPath != nil {
		logger.PduSessLog.Infoln("Direct Forwarding Path Available")
		ctx.DLForwardingType = DirectForwarding
	} else {
		logger.PduSessLog.Infoln("Direct Forwarding Path Unavailable")
		ctx.DLForwardingType = IndirectForwarding
	}

	if err != nil {
		return err
	}
	return nil
}

func HandleHandoverRequestAcknowledgeTransfer(b []byte, ctx *SMContext) (err error) {
	handoverRequestAcknowledgeTransfer := ngapType.HandoverRequestAcknowledgeTransfer{}

	err = aper.UnmarshalWithParams(b, &handoverRequestAcknowledgeTransfer, "valueExt")

	if err != nil {
		return err
	}

	DLNGUUPGTPTunnel := handoverRequestAcknowledgeTransfer.DLNGUUPTNLInformation.GTPTunnel

	ctx.Tunnel.UpdateANInformation(
		DLNGUUPGTPTunnel.TransportLayerAddress.Value.Bytes,
		binary.BigEndian.Uint32(DLNGUUPGTPTunnel.GTPTEID.Value))

	DLForwardingInfo := handoverRequestAcknowledgeTransfer.DLForwardingUPTNLInformation

	if DLForwardingInfo == nil {
		return errors.New("DL Forwarding Info not provision")
	}

	if ctx.DLForwardingType == IndirectForwarding {
		DLForwardingGTPTunnel := DLForwardingInfo.GTPTunnel

		ctx.IndirectForwardingTunnel = NewDataPath()
		ctx.IndirectForwardingTunnel.FirstDPNode = NewDataPathNode()
		ctx.IndirectForwardingTunnel.FirstDPNode.UPF = ctx.Tunnel.DataPathPool.GetDefaultPath().FirstDPNode.UPF
		ctx.IndirectForwardingTunnel.FirstDPNode.UpLinkTunnel = &GTPTunnel{}

		ANUPF := ctx.IndirectForwardingTunnel.FirstDPNode.UPF

		var indirectFowardingPDR *PDR

		if pdr, err := ANUPF.AddPDR(); err != nil {
			return err
		} else {
			indirectFowardingPDR = pdr
		}

		originPDR := ctx.Tunnel.DataPathPool.GetDefaultPath().FirstDPNode.UpLinkTunnel.PDR

		if teid, err := ANUPF.GenerateTEID(); err != nil {
			return err
		} else {
			ctx.IndirectForwardingTunnel.FirstDPNode.UpLinkTunnel.TEID = teid
			ctx.IndirectForwardingTunnel.FirstDPNode.UpLinkTunnel.PDR = indirectFowardingPDR
			indirectFowardingPDR.PDI.LocalFTeid = &pfcpType.FTEID{
				V4:          originPDR.PDI.LocalFTeid.V4,
				Teid:        ctx.IndirectForwardingTunnel.FirstDPNode.UpLinkTunnel.TEID,
				Ipv4Address: originPDR.PDI.LocalFTeid.Ipv4Address,
			}
			indirectFowardingPDR.OuterHeaderRemoval = &pfcpType.OuterHeaderRemoval{
				OuterHeaderRemovalDescription: pfcpType.OuterHeaderRemovalGtpUUdpIpv4,
			}

			indirectFowardingPDR.FAR.ApplyAction = pfcpType.ApplyAction{
				Forw: true,
			}
			indirectFowardingPDR.FAR.ForwardingParameters = &ForwardingParameters{
				DestinationInterface: pfcpType.DestinationInterface{
					InterfaceValue: pfcpType.DestinationInterfaceAccess,
				},
				OuterHeaderCreation: &pfcpType.OuterHeaderCreation{
					OuterHeaderCreationDescription: pfcpType.OuterHeaderCreationGtpUUdpIpv4,
					Teid:                           binary.BigEndian.Uint32(DLForwardingGTPTunnel.GTPTEID.Value),
					Ipv4Address:                    DLForwardingGTPTunnel.TransportLayerAddress.Value.Bytes,
				},
			}
		}
	} else if ctx.DLForwardingType == DirectForwarding {
		ctx.DLDirectForwardingTunnel = DLForwardingInfo
	}

	return nil
}
