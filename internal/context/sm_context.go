package context

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/antihax/optional"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"bitbucket.org/free5gc-team/nas/nasConvert"
	"bitbucket.org/free5gc-team/nas/nasMessage"
	"bitbucket.org/free5gc-team/ngap/ngapType"
	"bitbucket.org/free5gc-team/openapi"
	"bitbucket.org/free5gc-team/openapi/Namf_Communication"
	"bitbucket.org/free5gc-team/openapi/Nnrf_NFDiscovery"
	"bitbucket.org/free5gc-team/openapi/Npcf_SMPolicyControl"
	"bitbucket.org/free5gc-team/openapi/models"
	"bitbucket.org/free5gc-team/pfcp/pfcpType"
	"bitbucket.org/free5gc-team/smf/internal/logger"
	"bitbucket.org/free5gc-team/util/idgenerator"
)

var (
	smContextPool    sync.Map
	canonicalRef     sync.Map
	seidSMContextMap sync.Map
)

type DLForwardingType int

const (
	IndirectForwarding DLForwardingType = iota
	DirectForwarding
	NoForwarding
)

var smContextCount uint64

type SMContextState uint32

const (
	InActive SMContextState = iota
	ActivePending
	Active
	InActivePending
	ModificationPending
	PFCPModification
)

func init() {
}

func GetSMContextCount() uint64 {
	atomic.AddUint64(&smContextCount, 1)
	return smContextCount
}

type SMContext struct {
	*models.SmContextCreateData

	Ref string

	LocalSEID  uint64
	RemoteSEID uint64

	UnauthenticatedSupi bool

	Pei          string
	Identifier   string
	PDUSessionID int32

	UpCnxState models.UpCnxState

	HoState models.HoState

	PDUAddress             net.IP
	UseStaticIP            bool
	SelectedPDUSessionType uint8

	DnnConfiguration models.DnnConfiguration

	SMPolicyID string

	// Handover related
	DLForwardingType         DLForwardingType
	DLDirectForwardingTunnel *ngapType.UPTransportLayerInformation
	IndirectForwardingTunnel *DataPath

	// UP Security support TS 29.502 R16 6.1.6.2.39
	UpSecurity                                                     *models.UpSecurity
	MaximumDataRatePerUEForUserPlaneIntegrityProtectionForUpLink   models.MaxIntegrityProtectedDataRate
	MaximumDataRatePerUEForUserPlaneIntegrityProtectionForDownLink models.MaxIntegrityProtectedDataRate
	// SMF verified UP security result of Xn-handover TS 33.501 6.6.1
	UpSecurityFromPathSwitchRequestSameAsLocalStored bool

	// Client
	SMPolicyClient      *Npcf_SMPolicyControl.APIClient
	CommunicationClient *Namf_Communication.APIClient

	AMFProfile         models.NfProfile
	SelectedPCFProfile models.NfProfile
	SmStatusNotifyUri  string

	Tunnel      *UPTunnel
	SelectedUPF *UPNode
	BPManager   *BPManager
	// NodeID(string form) to PFCP Session Context
	PFCPContext                         map[string]*PFCPSessionContext
	sbiPFCPCommunicationChan            chan PFCPSessionResponseStatus
	PendingUPF                          PendingUPF
	PDUSessionRelease_DUE_TO_DUP_PDU_ID bool

	DNNInfo *SnssaiSmfDnnInfo

	// SM Policy related
	PCCRules            map[string]*PCCRule
	SessionRules        map[string]*SessionRule
	TrafficControlDatas map[string]*TrafficControlData

	SelectedSessionRuleID string

	// QoS
	QoSRuleIDGenerator      *idgenerator.IDGenerator
	PacketFilterIDGenerator *idgenerator.IDGenerator
	PCCRuleIDToQoSRuleID    map[string]uint8
	PacketFilterIDToNASPFID map[string]uint8

	// NAS
	Pti                     uint8
	EstAcceptCause5gSMValue uint8

	// PCO Related
	ProtocolConfigurationOptions *ProtocolConfigurationOptions

	// State
	state SMContextState

	// Loggers
	Log *logrus.Entry

	// 5GSM Timers
	// T3591 is PDU SESSION MODIFICATION COMMAND timer
	T3591 *Timer
	// T3592 is PDU SESSION RELEASE COMMAND timer
	T3592 *Timer

	// lock
	SMLock      sync.Mutex
	SMLockTimer *time.Timer
}

func canonicalName(id string, pduSessID int32) string {
	return fmt.Sprintf("%s-%d", id, pduSessID)
}

func ResolveRef(id string, pduSessID int32) (string, error) {
	if value, ok := canonicalRef.Load(canonicalName(id, pduSessID)); ok {
		ref := value.(string)
		return ref, nil
	} else {
		return "", fmt.Errorf("UE[%s] - PDUSessionID[%d] not found in SMContext", id, pduSessID)
	}
}

func NewSMContext(id string, pduSessID int32) *SMContext {
	smContext := new(SMContext)
	// Create Ref and identifier
	smContext.Ref = uuid.New().URN()
	smContextPool.Store(smContext.Ref, smContext)
	canonicalRef.Store(canonicalName(id, pduSessID), smContext.Ref)

	smContext.Log = logger.PduSessLog.WithFields(logrus.Fields{
		logger.FieldSupi:         id,
		logger.FieldPDUSessionID: fmt.Sprintf("%d", pduSessID),
	})

	smContext.SetState(InActive)
	smContext.Identifier = id
	smContext.PDUSessionID = pduSessID
	smContext.PFCPContext = make(map[string]*PFCPSessionContext)
	smContext.LocalSEID = GetSMContextCount()

	// initialize SM Policy Data
	smContext.PCCRules = make(map[string]*PCCRule)
	smContext.SessionRules = make(map[string]*SessionRule)
	smContext.TrafficControlDatas = make(map[string]*TrafficControlData)
	smContext.sbiPFCPCommunicationChan = make(chan PFCPSessionResponseStatus, 1)

	smContext.ProtocolConfigurationOptions = &ProtocolConfigurationOptions{}

	smContext.BPManager = NewBPManager(id)
	smContext.Tunnel = NewUPTunnel()
	smContext.PendingUPF = make(PendingUPF)

	smContext.QoSRuleIDGenerator = idgenerator.NewGenerator(1, 255)
	smContext.PacketFilterIDGenerator = idgenerator.NewGenerator(1, 255)
	smContext.PCCRuleIDToQoSRuleID = make(map[string]uint8)
	smContext.PacketFilterIDToNASPFID = make(map[string]uint8)

	return smContext
}

//*** add unit test ***//
func GetSMContextByRef(ref string) *SMContext {
	var smCtx *SMContext
	if value, ok := smContextPool.Load(ref); ok {
		smCtx = value.(*SMContext)
	}
	return smCtx
}

func GetSMContextById(id string, pduSessID int32) *SMContext {
	var smCtx *SMContext
	ref, err := ResolveRef(id, pduSessID)
	if err != nil {
		return nil
	}
	if value, ok := smContextPool.Load(ref); ok {
		smCtx = value.(*SMContext)
	}
	return smCtx
}

//*** add unit test ***//
func RemoveSMContext(ref string) {
	var smContext *SMContext
	if value, ok := smContextPool.Load(ref); ok {
		smContext = value.(*SMContext)
	} else {
		return
	}

	if smContext.SelectedUPF != nil {
		logger.PduSessLog.Infof("UE[%s] PDUSessionID[%d] Release IP[%s]",
			smContext.Supi, smContext.PDUSessionID, smContext.PDUAddress.String())
		GetUserPlaneInformation().
			ReleaseUEIP(smContext.SelectedUPF, smContext.PDUAddress, smContext.UseStaticIP)
		smContext.SelectedUPF = nil
	}

	for _, pfcpSessionContext := range smContext.PFCPContext {
		seidSMContextMap.Delete(pfcpSessionContext.LocalSEID)
	}

	smContextPool.Delete(ref)
	canonicalRef.Delete(canonicalName(smContext.Supi, smContext.PDUSessionID))
	smContext.Log.Infof("smContext[%s] is deleted from pool", ref)
}

//*** add unit test ***//
func GetSMContextBySEID(SEID uint64) *SMContext {
	if value, ok := seidSMContextMap.Load(SEID); ok {
		smContext := value.(*SMContext)
		return smContext
	}
	return nil
}

func (smContext *SMContext) WaitPFCPCommunicationStatus() PFCPSessionResponseStatus {
	var PFCPResponseStatus PFCPSessionResponseStatus
	select {
	case <-time.After(time.Second * 4):
		PFCPResponseStatus = SessionUpdateFailed
	case PFCPResponseStatus = <-smContext.sbiPFCPCommunicationChan:
	}
	return PFCPResponseStatus
}

func (smContext *SMContext) SendPFCPCommunicationStatus(status PFCPSessionResponseStatus) {
	smContext.sbiPFCPCommunicationChan <- status
}

func (smContext *SMContext) BuildCreatedData() *models.SmContextCreatedData {
	return &models.SmContextCreatedData{
		SNssai: smContext.SNssai,
	}
}

func (smContext *SMContext) SetState(state SMContextState) {
	oldState := SMContextState(atomic.LoadUint32((*uint32)(&smContext.state)))

	atomic.StoreUint32((*uint32)(&smContext.state), uint32(state))
	smContext.Log.Tracef("State[%s] -> State[%s]", oldState, state)
}

func (smContext *SMContext) CheckState(state SMContextState) bool {
	curState := SMContextState(atomic.LoadUint32((*uint32)(&smContext.state)))
	check := curState == state
	if !check {
		smContext.Log.Warnf("Unexpected state, expect: [%s], actual:[%s]", state, curState)
	}
	return check
}

func (smContext *SMContext) State() SMContextState {
	return SMContextState(atomic.LoadUint32((*uint32)(&smContext.state)))
}

func (smContext *SMContext) PDUAddressToNAS() ([12]byte, uint8) {
	var addr [12]byte
	var addrLen uint8
	copy(addr[:], smContext.PDUAddress)
	switch smContext.SelectedPDUSessionType {
	case nasMessage.PDUSessionTypeIPv4:
		addrLen = 4 + 1
	case nasMessage.PDUSessionTypeIPv6:
	case nasMessage.PDUSessionTypeIPv4IPv6:
		addrLen = 12 + 1
	}
	return addr, addrLen
}

// PCFSelection will select PCF for this SM Context
func (smContext *SMContext) PCFSelection() error {
	// Send NFDiscovery for find PCF
	localVarOptionals := Nnrf_NFDiscovery.SearchNFInstancesParamOpts{}

	if SMF_Self().Locality != "" {
		localVarOptionals.PreferredLocality = optional.NewString(SMF_Self().Locality)
	}

	rep, res, err := SMF_Self().
		NFDiscoveryClient.
		NFInstancesStoreApi.
		SearchNFInstances(context.TODO(), models.NfType_PCF, models.NfType_SMF, &localVarOptionals)
	if err != nil {
		return err
	}
	defer func() {
		if rspCloseErr := res.Body.Close(); rspCloseErr != nil {
			logger.PduSessLog.Errorf("SmfEventExposureNotification response body cannot close: %+v", rspCloseErr)
		}
	}()

	if res != nil {
		if status := res.StatusCode; status != http.StatusOK {
			apiError := err.(openapi.GenericOpenAPIError)
			problemDetails := apiError.Model().(models.ProblemDetails)

			logger.CtxLog.Warningf("NFDiscovery PCF return status: %d\n", status)
			logger.CtxLog.Warningf("Detail: %v\n", problemDetails.Title)
		}
	}

	// Select PCF from available PCF

	smContext.SelectedPCFProfile = rep.NfInstances[0]

	// Create SMPolicyControl Client for this SM Context
	for _, service := range *smContext.SelectedPCFProfile.NfServices {
		if service.ServiceName == models.ServiceName_NPCF_SMPOLICYCONTROL {
			SmPolicyControlConf := Npcf_SMPolicyControl.NewConfiguration()
			SmPolicyControlConf.SetBasePath(service.ApiPrefix)
			smContext.SMPolicyClient = Npcf_SMPolicyControl.NewAPIClient(SmPolicyControlConf)
		}
	}

	return nil
}

func (smContext *SMContext) GetNodeIDByLocalSEID(seid uint64) pfcpType.NodeID {
	for _, pfcpCtx := range smContext.PFCPContext {
		if pfcpCtx.LocalSEID == seid {
			return pfcpCtx.NodeID
		}
	}

	return pfcpType.NodeID{}
}

func (smContext *SMContext) AllocateLocalSEIDForUPPath(path UPPath) {
	for _, upNode := range path {
		NodeIDtoIP := upNode.NodeID.ResolveNodeIdToIp().String()
		if _, exist := smContext.PFCPContext[NodeIDtoIP]; !exist {
			allocatedSEID := AllocateLocalSEID()

			smContext.PFCPContext[NodeIDtoIP] = &PFCPSessionContext{
				PDRs:      make(map[uint16]*PDR),
				NodeID:    upNode.NodeID,
				LocalSEID: allocatedSEID,
			}

			seidSMContextMap.Store(allocatedSEID, smContext)
		}
	}
}

func (smContext *SMContext) AllocateLocalSEIDForDataPath(dataPath *DataPath) {
	logger.PduSessLog.Traceln("In AllocateLocalSEIDForDataPath")
	for curDataPathNode := dataPath.FirstDPNode; curDataPathNode != nil; curDataPathNode = curDataPathNode.Next() {
		NodeIDtoIP := curDataPathNode.UPF.NodeID.ResolveNodeIdToIp().String()
		logger.PduSessLog.Traceln("NodeIDtoIP: ", NodeIDtoIP)
		if _, exist := smContext.PFCPContext[NodeIDtoIP]; !exist {
			allocatedSEID := AllocateLocalSEID()
			smContext.PFCPContext[NodeIDtoIP] = &PFCPSessionContext{
				PDRs:      make(map[uint16]*PDR),
				NodeID:    curDataPathNode.UPF.NodeID,
				LocalSEID: allocatedSEID,
			}

			seidSMContextMap.Store(allocatedSEID, smContext)
		}
	}
}

func (smContext *SMContext) PutPDRtoPFCPSession(nodeID pfcpType.NodeID, pdr *PDR) error {
	NodeIDtoIP := nodeID.ResolveNodeIdToIp().String()
	if pfcpSessCtx, exist := smContext.PFCPContext[NodeIDtoIP]; exist {
		pfcpSessCtx.PDRs[pdr.PDRID] = pdr
	} else {
		return fmt.Errorf("Can't find PFCPContext[%s] to put PDR(%d)", NodeIDtoIP, pdr.PDRID)
	}
	return nil
}

func (c *SMContext) findPSAandAllocUeIP(param *UPFSelectionParams) error {
	c.Log.Traceln("findPSAandAllocUeIP")
	if param == nil {
		return fmt.Errorf("UPFSelectionParams is nil")
	}

	upInfo := GetUserPlaneInformation()
	if SMF_Self().ULCLSupport && CheckUEHasPreConfig(c.Supi) {
		groupName := GetULCLGroupNameFromSUPI(c.Supi)
		preConfigPathPool := GetUEDefaultPathPool(groupName)
		if preConfigPathPool != nil {
			selectedUPFName := ""
			selectedUPFName, c.PDUAddress, c.UseStaticIP =
				preConfigPathPool.SelectUPFAndAllocUEIPForULCL(
					upInfo, param)
			c.SelectedUPF = GetUserPlaneInformation().UPFs[selectedUPFName]
		}
	} else {
		c.SelectedUPF, c.PDUAddress, c.UseStaticIP =
			GetUserPlaneInformation().SelectUPFAndAllocUEIP(param)
		c.Log.Infof("Allocated PDUAdress[%s]", c.PDUAddress.String())
	}
	if c.PDUAddress == nil {
		return fmt.Errorf("fail to allocate PDU address, Selection Parameter: %s",
			param.String())
	}
	return nil
}

func (c *SMContext) SelectDefaultDataPath() error {
	param := &UPFSelectionParams{
		Dnn: c.Dnn,
		SNssai: &SNssai{
			Sst: c.SNssai.Sst,
			Sd:  c.SNssai.Sd,
		},
	}

	if len(c.DnnConfiguration.StaticIpAddress) > 0 {
		staticIPConfig := c.DnnConfiguration.StaticIpAddress[0]
		if staticIPConfig.Ipv4Addr != "" {
			param.PDUAddress = net.ParseIP(staticIPConfig.Ipv4Addr).To4()
		}
	}

	if err := c.findPSAandAllocUeIP(param); err != nil {
		return err
	}

	var defaultPath *DataPath
	if SMF_Self().ULCLSupport && CheckUEHasPreConfig(c.Supi) {
		c.Log.Infof("Has pre-config route")
		uePreConfigPaths := GetUEPreConfigPaths(c.Supi, c.SelectedUPF.Name)
		c.Tunnel.DataPathPool = uePreConfigPaths.DataPathPool
		c.Tunnel.PathIDGenerator = uePreConfigPaths.PathIDGenerator
		defaultPath = c.Tunnel.DataPathPool.GetDefaultPath()
	} else {
		// UE has no pre-config path.
		// Use default route
		c.Log.Infof("Has no pre-config route")
		defaultUPPath := GetUserPlaneInformation().GetDefaultUserPlanePathByDNNAndUPF(
			param, c.SelectedUPF)
		defaultPath = GenerateDataPath(defaultUPPath)
		if defaultPath != nil {
			defaultPath.IsDefaultPath = true
			c.Tunnel.AddDataPath(defaultPath)
		}
	}

	if defaultPath == nil {
		return fmt.Errorf("Data Path not found, Selection Parameter: %s",
			param.String())
	}
	defaultPath.ActivateTunnelAndPDR(c, 255)
	return nil
}

func (c *SMContext) CreatePccRuleDataPath(pccRule *PCCRule,
	tcData *TrafficControlData) {
	var targetDNAI string
	if tcData != nil && len(tcData.RouteToLocs) > 0 {
		targetDNAI = tcData.RouteToLocs[0].Dnai
	}
	param := &UPFSelectionParams{
		Dnn: c.Dnn,
		SNssai: &SNssai{
			Sst: c.SNssai.Sst,
			Sd:  c.SNssai.Sd,
		},
		Dnai: targetDNAI,
	}
	createdUpPath := GetUserPlaneInformation().GetDefaultUserPlanePathByDNN(param)
	createdDataPath := GenerateDataPath(createdUpPath)
	if createdDataPath != nil {
		createdDataPath.ActivateTunnelAndPDR(c, 255-uint32(pccRule.Precedence))
		c.Tunnel.AddDataPath(createdDataPath)
	}

	pccRule.Datapath = createdDataPath
}

func (smContext *SMContext) RemovePDRfromPFCPSession(nodeID pfcpType.NodeID, pdr *PDR) {
	NodeIDtoIP := nodeID.ResolveNodeIdToIp().String()
	pfcpSessCtx := smContext.PFCPContext[NodeIDtoIP]
	delete(pfcpSessCtx.PDRs, pdr.PDRID)
}

func (smContext *SMContext) IsAllowedPDUSessionType(requestedPDUSessionType uint8) error {
	dnnPDUSessionType := smContext.DnnConfiguration.PduSessionTypes
	if dnnPDUSessionType == nil {
		return fmt.Errorf("this SMContext[%s] has no subscription pdu session type info", smContext.Ref)
	}

	allowIPv4 := false
	allowIPv6 := false
	allowEthernet := false

	for _, allowedPDUSessionType := range smContext.DnnConfiguration.PduSessionTypes.AllowedSessionTypes {
		switch allowedPDUSessionType {
		case models.PduSessionType_IPV4:
			allowIPv4 = true
		case models.PduSessionType_IPV6:
			allowIPv6 = true
		case models.PduSessionType_IPV4_V6:
			allowIPv4 = true
			allowIPv6 = true
		case models.PduSessionType_ETHERNET:
			allowEthernet = true
		}
	}

	supportedPDUSessionType := SMF_Self().SupportedPDUSessionType
	switch supportedPDUSessionType {
	case "IPv4":
		if !allowIPv4 {
			return fmt.Errorf("No SupportedPDUSessionType[%q] in DNN[%s] configuration", supportedPDUSessionType, smContext.Dnn)
		}
	case "IPv6":
		if !allowIPv6 {
			return fmt.Errorf("No SupportedPDUSessionType[%q] in DNN[%s] configuration", supportedPDUSessionType, smContext.Dnn)
		}
	case "IPv4v6":
		if !allowIPv4 && !allowIPv6 {
			return fmt.Errorf("No SupportedPDUSessionType[%q] in DNN[%s] configuration", supportedPDUSessionType, smContext.Dnn)
		}
	case "Ethernet":
		if !allowEthernet {
			return fmt.Errorf("No SupportedPDUSessionType[%q] in DNN[%s] configuration", supportedPDUSessionType, smContext.Dnn)
		}
	}

	smContext.EstAcceptCause5gSMValue = 0
	switch nasConvert.PDUSessionTypeToModels(requestedPDUSessionType) {
	case models.PduSessionType_IPV4:
		if allowIPv4 {
			smContext.SelectedPDUSessionType = nasConvert.ModelsToPDUSessionType(models.PduSessionType_IPV4)
		} else {
			return fmt.Errorf("PduSessionType_IPV4 is not allowed in DNN[%s] configuration", smContext.Dnn)
		}
	case models.PduSessionType_IPV6:
		if allowIPv6 {
			smContext.SelectedPDUSessionType = nasConvert.ModelsToPDUSessionType(models.PduSessionType_IPV6)
		} else {
			return fmt.Errorf("PduSessionType_IPV6 is not allowed in DNN[%s] configuration", smContext.Dnn)
		}
	case models.PduSessionType_IPV4_V6:
		if allowIPv4 && allowIPv6 {
			smContext.SelectedPDUSessionType = nasConvert.ModelsToPDUSessionType(models.PduSessionType_IPV4_V6)
		} else if allowIPv4 {
			smContext.SelectedPDUSessionType = nasConvert.ModelsToPDUSessionType(models.PduSessionType_IPV4)
			smContext.EstAcceptCause5gSMValue = nasMessage.Cause5GSMPDUSessionTypeIPv4OnlyAllowed
		} else if allowIPv6 {
			smContext.SelectedPDUSessionType = nasConvert.ModelsToPDUSessionType(models.PduSessionType_IPV6)
			smContext.EstAcceptCause5gSMValue = nasMessage.Cause5GSMPDUSessionTypeIPv6OnlyAllowed
		} else {
			return fmt.Errorf("PduSessionType_IPV4_V6 is not allowed in DNN[%s] configuration", smContext.Dnn)
		}
	case models.PduSessionType_ETHERNET:
		if allowEthernet {
			smContext.SelectedPDUSessionType = nasConvert.ModelsToPDUSessionType(models.PduSessionType_ETHERNET)
		} else {
			return fmt.Errorf("PduSessionType_ETHERNET is not allowed in DNN[%s] configuration", smContext.Dnn)
		}
	default:
		return fmt.Errorf("Requested PDU Sesstion type[%d] is not supported", requestedPDUSessionType)
	}
	return nil
}

// SM Policy related operation

// SelectedSessionRule - return the SMF selected session rule for this SM Context
func (c *SMContext) SelectedSessionRule() *SessionRule {
	if c.SelectedSessionRuleID == "" {
		return nil
	}
	return c.SessionRules[c.SelectedSessionRuleID]
}

func (c *SMContext) UpdateSessionRules(sessRules map[string]*models.SessionRule) error {
	for id, r := range sessRules {
		if r == nil {
			c.Log.Debugf("Delete SessionRule[%s]", id)
			delete(c.SessionRules, id)
		} else {
			if origRule, ok := c.SessionRules[id]; ok {
				c.Log.Debugf("Modify SessionRule[%s]: %+v", id, r)
				origRule.SessionRule = r
			} else {
				c.Log.Debugf("Install SessionRule[%s]: %+v", id, r)
				c.SessionRules[id] = NewSessionRule(r)
			}
		}
	}

	if len(c.SessionRules) == 0 {
		return fmt.Errorf("UpdateSessionRules: no session rule")
	}

	if c.SelectedSessionRuleID == "" ||
		c.SessionRules[c.SelectedSessionRuleID] == nil {
		// No active session rule, randomly choose a session rule to activate
		for id := range c.SessionRules {
			c.SelectedSessionRuleID = id
			break
		}
	}
	return nil

	// TODO: need update default data path if SessionRule changed
}

func (c *SMContext) UpdateTrafficControlDatas(traffContDecs map[string]*models.TrafficControlData) {
	for id, tc := range traffContDecs {
		if tc == nil {
			c.Log.Debugf("Delete TrafficControlData[%s]", id)
			delete(c.TrafficControlDatas, id)
		} else {
			if origData, ok := c.TrafficControlDatas[id]; ok {
				c.Log.Debugf("Modify TrafficControlData[%s]: %+v", id, tc)
				origData.TrafficControlData = tc
			} else {
				c.Log.Debugf("Install TrafficControlData[%s]: %+v", id, tc)
				c.TrafficControlDatas[id] = NewTrafficControlData(tc)
			}
		}
	}
}

func (smContext *SMContext) StopT3591() {
	if smContext.T3591 != nil {
		smContext.T3591.Stop()
		smContext.T3591 = nil
	}
}

func (smContext *SMContext) StopT3592() {
	if smContext.T3592 != nil {
		smContext.T3592.Stop()
		smContext.T3592 = nil
	}
}

func (smContextState SMContextState) String() string {
	switch smContextState {
	case InActive:
		return "InActive"
	case ActivePending:
		return "ActivePending"
	case Active:
		return "Active"
	case InActivePending:
		return "InActivePending"
	case ModificationPending:
		return "ModificationPending"
	case PFCPModification:
		return "PFCPModification"
	default:
		return "Unknown State"
	}
}
