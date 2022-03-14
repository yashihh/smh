package context

import (
	"fmt"

	"bitbucket.org/free5gc-team/openapi/models"
	"bitbucket.org/free5gc-team/smf/internal/logger"
	"bitbucket.org/free5gc-team/smf/pkg/factory"
	"bitbucket.org/free5gc-team/util/flowdesc"
)

// PCCRule - Policy and Charging Rule
type PCCRule struct {
	*models.PccRule

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
	ulIPFilterRule := flowdesc.NewIPFilterRule()

	err := flowdesc.Decode(dlFlowDesc, ulIPFilterRule)
	if err != nil {
		return ""
	}

	ulIPFilterRule.SwapSourceAndDestination()
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
