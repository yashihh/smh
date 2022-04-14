package context

import (
	"fmt"
	"net"
	"strconv"

	"bitbucket.org/free5gc-team/nas/nasType"
	"bitbucket.org/free5gc-team/openapi/models"
	"bitbucket.org/free5gc-team/smf/internal/logger"
	"bitbucket.org/free5gc-team/smf/pkg/factory"
	"bitbucket.org/free5gc-team/util/flowdesc"
)

// PCCRule - Policy and Charging Rule
type PCCRule struct {
	*models.PccRule
	QFI uint8
	// related Data
	Datapath *DataPath
}

// NewPCCRule - create PCC rule from OpenAPI models
func NewPCCRule(mPcc *models.PccRule) *PCCRule {
	if mPcc == nil {
		return nil
	}

	return &PCCRule{
		PccRule: mPcc,
	}
}

func (r *PCCRule) FlowDescription() string {
	if len(r.FlowInfos) > 0 {
		// now 1 pcc rule only maps to 1 FlowInfo
		return r.FlowInfos[0].FlowDescription
	}
	return ""
}

func (r *PCCRule) RefQosDataID() string {
	if len(r.RefQosData) > 0 {
		// now 1 pcc rule only maps to 1 QoS data
		return r.RefQosData[0]
	}
	return ""
}

func (r *PCCRule) RefTcDataID() string {
	if len(r.RefTcData) > 0 {
		// now 1 pcc rule only maps to 1 Traffic Control data
		return r.RefTcData[0]
	}
	return ""
}

func (r *PCCRule) UpdateDataPathFlowDescription(dlFlowDesc string) error {
	if r.Datapath == nil {
		return fmt.Errorf("pcc[%s]: no data path", r.PccRuleId)
	}

	if dlFlowDesc == "" {
		return fmt.Errorf("pcc[%s]: no flow description", r.PccRuleId)
	}
	ulFlowDesc := getUplinkFlowDescription(dlFlowDesc)
	if ulFlowDesc == "" {
		return fmt.Errorf("pcc[%s]: uplink flow description parsing error", r.PccRuleId)
	}
	r.Datapath.UpdateFlowDescription(ulFlowDesc, dlFlowDesc)
	return nil
}

func getUplinkFlowDescription(dlFlowDesc string) string {
	ulIPFilterRule, err := flowdesc.Decode(dlFlowDesc)
	if err != nil {
		return ""
	}

	ulIPFilterRule.SwapSrcAndDst()
	ulFlowDesc, err := flowdesc.Encode(ulIPFilterRule)
	if err != nil {
		return ""
	}
	return ulFlowDesc
}

func (r *PCCRule) AddDataPathForwardingParameters(c *SMContext,
	tgtRoute *models.RouteToLocation) {
	if tgtRoute == nil {
		return
	}

	if r.Datapath == nil {
		logger.CtxLog.Warnf("AddDataPathForwardingParameters pcc[%s]: no data path", r.PccRuleId)
		return
	}

	var routeProf factory.RouteProfile
	routeProfExist := false
	// specify N6 routing information
	if tgtRoute.RouteProfId != "" {
		routeProf, routeProfExist = factory.UERoutingConfig.RouteProf[factory.RouteProfID(tgtRoute.RouteProfId)]
		if !routeProfExist {
			logger.CtxLog.Warnf("Route Profile ID [%s] is not support", tgtRoute.RouteProfId)
			return
		}
	}
	r.Datapath.AddForwardingParameters(routeProf.ForwardingPolicyID,
		c.Tunnel.DataPathPool.GetDefaultPath().FirstDPNode.GetUpLinkPDR().PDI.LocalFTeid.Teid)
}

func (r *PCCRule) AddDataPathQoSData(qos *models.QosData) {
	if r.Datapath == nil {
		logger.CtxLog.Warnf("AddDataPathQoSData pcc[%s]: no data path", r.PccRuleId)
		return
	}
	r.Datapath.AddQoS(qos)
}

func (r *PCCRule) BuildNasQoSRule(smCtx *SMContext,
	opCode nasType.QoSRuleOperationCode) (*nasType.QoSRule, error) {
	rule := nasType.QoSRule{}
	rule.Operation = nasType.OperationCodeCreateNewQoSRule
	rule.Precedence = uint8(r.Precedence)
	pfList := make(nasType.PacketFilterList, 0)
	for _, flowInfo := range r.FlowInfos {
		if pfs, err := BuildNASPacketFiltersFromFlowInformation(&flowInfo, smCtx); err != nil {
			logger.CtxLog.Warnf("BuildNasQoSRule: Build packet filter fail: %s\n", err)
			continue
		} else {
			pfList = append(pfList, pfs...)
		}
	}
	rule.PacketFilterList = pfList
	rule.QFI = r.QFI

	return &rule, nil
}

func BuildNASPacketFiltersFromFlowInformation(pfInfo *models.FlowInformation,
	smCtx *SMContext) ([]nasType.PacketFilter, error) {
	var pfList []nasType.PacketFilter
	var err error

	ipFilterRule := flowdesc.NewIPFilterRule()
	if pfInfo.FlowDescription != "" {
		ipFilterRule, err = flowdesc.Decode(pfInfo.FlowDescription)
		if err != nil {
			return nil, fmt.Errorf("parse packet filter content fail: %s", err)
		}
	}

	srcPorts := ipFilterRule.SrcPorts
	dstPorts := ipFilterRule.DstPorts
	// append an port range to avoid ipFilterRule.SrcPorts/ipFilterRule.DstPorts length is 0
	// if length is 0 for loop will not be executed
	if len(ipFilterRule.SrcPorts) == 0 {
		ipFilterRule.SrcPorts = append(ipFilterRule.SrcPorts, flowdesc.PortRange{
			Start: 0,
			End:   0,
		})
	}
	if len(ipFilterRule.DstPorts) == 0 {
		ipFilterRule.DstPorts = append(ipFilterRule.DstPorts, flowdesc.PortRange{
			Start: 0,
			End:   0,
		})
	}
	for _, srcP := range srcPorts {
		for _, dstP := range dstPorts {
			pf := new(nasType.PacketFilter)

			if packetFilterID, err := smCtx.PacketFilterIDGenerator.Allocate(); err != nil {
				return nil, err
			} else {
				pf.Identifier = uint8(packetFilterID)
				smCtx.PacketFilterIDToNASPFID[pfInfo.PackFiltId] = uint8(packetFilterID)
			}

			switch pfInfo.FlowDirection {
			case models.FlowDirectionRm_DOWNLINK:
				pf.Direction = nasType.PacketFilterDirectionDownlink
			case models.FlowDirectionRm_UPLINK:
				pf.Direction = nasType.PacketFilterDirectionUplink
			case models.FlowDirectionRm_BIDIRECTIONAL:
				pf.Direction = nasType.PacketFilterDirectionBidirectional
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
			if ipFilterRule.Src != "any" {
				_, ipNet, err := net.ParseCIDR(ipFilterRule.Src)
				if err != nil {
					return nil, fmt.Errorf("parse IP fail: %s", err)
				}
				pfComponents = append(pfComponents, &nasType.PacketFilterIPv4RemoteAddress{
					Address: ipNet.IP.To4(),
					Mask:    ipNet.Mask,
				})
			}

			if ipFilterRule.Dst != "any" {
				_, ipNet, err := net.ParseCIDR(ipFilterRule.Dst)
				if err != nil {
					return nil, fmt.Errorf("parse IP fail: %s", err)
				}
				pfComponents = append(pfComponents, &nasType.PacketFilterIPv4LocalAddress{
					Address: ipNet.IP.To4(),
					Mask:    ipNet.Mask,
				})
			}

			if dstP.Start != dstP.End {
				pfComponents = append(pfComponents, &nasType.PacketFilterLocalPortRange{
					LowLimit:  dstP.Start,
					HighLimit: dstP.End,
				})
			} else if dstP.Start != 0 && dstP.End != 0 {
				pfComponents = append(pfComponents, &nasType.PacketFilterSingleLocalPort{
					Value: dstP.Start,
				})
			}

			if srcP.Start != srcP.End {
				pfComponents = append(pfComponents, &nasType.PacketFilterRemotePortRange{
					LowLimit:  srcP.Start,
					HighLimit: srcP.End,
				})
			} else if srcP.Start != 0 && dstP.End != 0 {
				pfComponents = append(pfComponents, &nasType.PacketFilterSingleRemotePort{
					Value: srcP.Start,
				})
			}

			if ipFilterRule.Proto != flowdesc.ProtocolNumberAny {
				pfComponents = append(pfComponents, &nasType.PacketFilterProtocolIdentifier{
					Value: ipFilterRule.Proto,
				})
			}

			if len(pfComponents) == 0 {
				pfComponents = append(pfComponents, &nasType.PacketFilterMatchAll{})
			}

			pf.Components = pfComponents
		}
	}

	return pfList, nil
}
