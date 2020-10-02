package context

import (
	"errors"
	"fmt"
	"math"
	"net"
	"reflect"
	"strconv"
	"sync"

	"github.com/google/uuid"

	"bitbucket.org/free5gc-team/idgenerator"
	"bitbucket.org/free5gc-team/nas/nasMessage"
	"bitbucket.org/free5gc-team/openapi/models"
	"bitbucket.org/free5gc-team/pfcp/pfcpType"
	"bitbucket.org/free5gc-team/pfcp/pfcpUdp"
	"bitbucket.org/free5gc-team/smf/factory"
	"bitbucket.org/free5gc-team/smf/logger"
)

var upfPool sync.Map

type UPTunnel struct {
	PathIDGenerator *idgenerator.IDGenerator
	DataPathPool    DataPathPool
	ANInformation   struct {
		IPAddress net.IP
		TEID      uint32
	}
}

type UPFStatus int

const (
	NotAssociated          UPFStatus = 0
	AssociatedSettingUp    UPFStatus = 1
	AssociatedSetUpSuccess UPFStatus = 2
)

type UPF struct {
	uuid         uuid.UUID
	NodeID       pfcpType.NodeID
	UPIPInfo     pfcpType.UserPlaneIPResourceInformation
	UPFStatus    UPFStatus
	SNssaiInfo   SnssaiInfo
	N3Interfaces []UPFInterfaceInfo
	N9Interfaces []UPFInterfaceInfo

	pdrPool sync.Map
	farPool sync.Map
	barPool sync.Map
	// qerPool sync.Map
	// urrPool        sync.Map
	pdrIDGenerator *idgenerator.IDGenerator
	farIDGenerator *idgenerator.IDGenerator
	barIDGenerator *idgenerator.IDGenerator
	urrIDGenerator *idgenerator.IDGenerator
	qerIDGenerator *idgenerator.IDGenerator
	teidGenerator  *idgenerator.IDGenerator
}

//UPFSelectionParams ... parameters for upf selection
type UPFSelectionParams struct {
	Dnn    string
	SNssai *SNssai
}

// UPFInterfaceInfo store the UPF interface information
type UPFInterfaceInfo struct {
	NetworkInstance       string
	IPv4EndPointAddresses []net.IP
	IPv6EndPointAddresses []net.IP
	EndpointFQDN          string
}

// NewUPFInterfaceInfo parse the InterfaceUpfInfoItem to generate UPFInterfaceInfo
func NewUPFInterfaceInfo(i *factory.InterfaceUpfInfoItem) *UPFInterfaceInfo {
	var interfaceInfo = new(UPFInterfaceInfo)

	interfaceInfo.IPv4EndPointAddresses = make([]net.IP, 0)
	interfaceInfo.IPv6EndPointAddresses = make([]net.IP, 0)

	logger.CtxLog.Infoln("Endpoints:", i.Endpoints)

	for _, endpoint := range i.Endpoints {
		eIP := net.ParseIP(endpoint)
		if eIP == nil {
			interfaceInfo.EndpointFQDN = endpoint
		} else if eIPv4 := eIP.To4(); eIPv4 == nil {
			interfaceInfo.IPv6EndPointAddresses = append(interfaceInfo.IPv6EndPointAddresses, eIP)
		} else {
			interfaceInfo.IPv4EndPointAddresses = append(interfaceInfo.IPv4EndPointAddresses, eIPv4)
		}
	}

	interfaceInfo.NetworkInstance = i.NetworkInstance

	return interfaceInfo
}

// IP returns the IP of the user plane IP information of the pduSessType
func (i *UPFInterfaceInfo) IP(pduSessType uint8) (net.IP, error) {

	if pduSessType == nasMessage.PDUSessionTypeIPv4 && len(i.IPv4EndPointAddresses) != 0 {
		return i.IPv4EndPointAddresses[0], nil
	}

	if pduSessType == nasMessage.PDUSessionTypeIPv6 && len(i.IPv6EndPointAddresses) != 0 {
		return i.IPv6EndPointAddresses[0], nil
	}

	if i.EndpointFQDN != "" {
		if resolvedAddr, err := net.ResolveIPAddr("ip", i.EndpointFQDN); err != nil {
			logger.CtxLog.Errorf("resolve addr [%s] failed", i.EndpointFQDN)
		} else {
			if pduSessType == nasMessage.PDUSessionTypeIPv4 {
				return resolvedAddr.IP.To4(), nil
			} else if pduSessType == nasMessage.PDUSessionTypeIPv6 {
				return resolvedAddr.IP.To16(), nil
			}
		}
	}

	return nil, errors.New("not matched ip address")
}

func (upfSelectionParams *UPFSelectionParams) String() string {
	Dnn := upfSelectionParams.Dnn
	SNssai := upfSelectionParams.SNssai

	str := ""
	if Dnn != "" {
		str += "\nDnn: " + upfSelectionParams.Dnn + "\n"
	}

	if SNssai != nil {
		str += "Sst: " + strconv.Itoa(int(upfSelectionParams.SNssai.Sst)) + "\n"
	}

	return str
}

// UUID return this UPF UUID (allocate by SMF in this time)
// Maybe allocate by UPF in future
func (upf *UPF) UUID() string {

	uuid := upf.uuid.String()
	return uuid
}

func NewUPTunnel() (tunnel *UPTunnel) {
	tunnel = &UPTunnel{
		DataPathPool:    make(DataPathPool),
		PathIDGenerator: idgenerator.NewGenerator(1, 2147483647),
	}

	return
}

func (upTunnel *UPTunnel) AddDataPath(dataPath *DataPath) {
	pathID, err := upTunnel.PathIDGenerator.Allocate()
	if err != nil {
		logger.CtxLog.Warnf("Allocate pathID error: %+v", err)
		return
	}

	upTunnel.DataPathPool[pathID] = dataPath
}

// NewUPF returns a new UPF context in SMF
func NewUPF(nodeID *pfcpType.NodeID, ifaces []factory.InterfaceUpfInfoItem) (upf *UPF) {
	upf = new(UPF)
	upf.uuid = uuid.New()

	upfPool.Store(upf.UUID(), upf)

	// Initialize context
	upf.UPFStatus = NotAssociated
	upf.NodeID = *nodeID
	upf.pdrIDGenerator = idgenerator.NewGenerator(1, math.MaxUint16)
	upf.farIDGenerator = idgenerator.NewGenerator(1, math.MaxUint32)
	upf.barIDGenerator = idgenerator.NewGenerator(1, math.MaxUint8)
	upf.qerIDGenerator = idgenerator.NewGenerator(1, math.MaxUint32)
	upf.urrIDGenerator = idgenerator.NewGenerator(1, math.MaxUint32)
	upf.teidGenerator = idgenerator.NewGenerator(1, math.MaxUint32)

	upf.N3Interfaces = make([]UPFInterfaceInfo, 0)
	upf.N9Interfaces = make([]UPFInterfaceInfo, 0)

	for _, iface := range ifaces {
		var upIface = NewUPFInterfaceInfo(&iface)

		switch iface.InterfaceType {
		case models.UpInterfaceType_N3:
			upf.N3Interfaces = append(upf.N3Interfaces, *upIface)
		case models.UpInterfaceType_N9:
			upf.N9Interfaces = append(upf.N9Interfaces, *upIface)
		}

	}

	return upf
}

// GetInterface return the UPFInterfaceInfo that match input cond
func (upf *UPF) GetInterface(interfaceType models.UpInterfaceType, dnn string) *UPFInterfaceInfo {
	switch interfaceType {
	case models.UpInterfaceType_N3:
		for i, iface := range upf.N3Interfaces {
			if iface.NetworkInstance == dnn {
				return &upf.N3Interfaces[i]
			}
		}
	case models.UpInterfaceType_N9:
		for i, iface := range upf.N9Interfaces {
			if iface.NetworkInstance == dnn {
				return &upf.N9Interfaces[i]
			}
		}
	}
	return nil
}

func (upf *UPF) GenerateTEID() (uint32, error) {

	if upf.UPFStatus != AssociatedSetUpSuccess {
		err := fmt.Errorf("this upf not associate with smf")
		return 0, err
	}

	var id uint32
	if tmpID, err := upf.teidGenerator.Allocate(); err != nil {
		return 0, err
	} else {
		id = uint32(tmpID)
	}

	return id, nil
}

func (upf *UPF) PFCPAddr() *net.UDPAddr {
	return &net.UDPAddr{
		IP:   upf.NodeID.ResolveNodeIdToIp(),
		Port: pfcpUdp.PFCP_PORT,
	}
}

func RetrieveUPFNodeByNodeID(nodeID pfcpType.NodeID) *UPF {
	var upf *UPF = nil
	upfPool.Range(func(key, value interface{}) bool {
		if reflect.DeepEqual(value.(*UPF).NodeID, nodeID) && upf.NodeID.NodeIdType != nodeID.NodeIdType &&
			(upf.NodeID.NodeIdType == pfcpType.NodeIdTypeFqdn || nodeID.NodeIdType == pfcpType.NodeIdTypeFqdn) {
			upfNodeIdIP := upf.NodeID.ResolveNodeIdToIp().To4()
			nodeIdIP := nodeID.ResolveNodeIdToIp().To4()
			logger.CtxLog.Tracef("RetrieveUPF - upfNodeIdIP:[%+v], nodeIdIP:[%+v]", upfNodeIdIP, nodeIdIP)
			if reflect.DeepEqual(upfNodeIdIP, nodeIdIP) {
				return false
			}
		} else if reflect.DeepEqual(upf.NodeID, nodeID) {
			return false
		}
		upf = nil
		return true
	})

	return upf
}

func RemoveUPFNodeByNodeID(nodeID pfcpType.NodeID) bool {
	upfID := ""
	upfPool.Range(func(key, value interface{}) bool {
		upfID = key.(string)
		upf := value.(*UPF)
		if upf.NodeID.NodeIdType != nodeID.NodeIdType &&
			(upf.NodeID.NodeIdType == pfcpType.NodeIdTypeFqdn || nodeID.NodeIdType == pfcpType.NodeIdTypeFqdn) {
			upfNodeIdIP := upf.NodeID.ResolveNodeIdToIp().To4()
			nodeIdIP := nodeID.ResolveNodeIdToIp().To4()
			logger.CtxLog.Tracef("RemoveUPF - upfNodeIdIP:[%+v], nodeIdIP:[%+v]", upfNodeIdIP, nodeIdIP)
			if reflect.DeepEqual(upfNodeIdIP, nodeIdIP) {
				return false
			}
		} else if reflect.DeepEqual(upf.NodeID, nodeID) {
			return false
		}
		upfID = ""
		return true
	})

	if upfID != "" {
		upfPool.Delete(upfID)
		return true
	}
	return false
}

func SelectUPFByDnn(Dnn string) *UPF {
	var upf *UPF
	upfPool.Range(func(key, value interface{}) bool {
		upf = value.(*UPF)
		if upf.UPIPInfo.Assoni && string(upf.UPIPInfo.NetworkInstance) == Dnn {
			return false
		}
		upf = nil
		return true
	})
	return upf
}

func (upf *UPF) GetUPFIP() string {
	upfIP := upf.NodeID.ResolveNodeIdToIp().String()
	return upfIP
}

func (upf *UPF) GetUPFID() string {
	upInfo := GetUserPlaneInformation()
	upfIP := upf.NodeID.ResolveNodeIdToIp().String()
	return upInfo.GetUPFIDByIP(upfIP)

}

func (upf *UPF) pdrID() (uint16, error) {
	if upf.UPFStatus != AssociatedSetUpSuccess {
		err := fmt.Errorf("this upf not associate with smf")
		return 0, err
	}

	var pdrID uint16
	if tmpID, err := upf.farIDGenerator.Allocate(); err != nil {
		return 0, err
	} else {
		pdrID = uint16(tmpID)
	}

	return pdrID, nil
}

func (upf *UPF) farID() (uint32, error) {
	if upf.UPFStatus != AssociatedSetUpSuccess {
		err := fmt.Errorf("this upf not associate with smf")
		return 0, err
	}

	var farID uint32
	if tmpID, err := upf.farIDGenerator.Allocate(); err != nil {
		return 0, err
	} else {
		farID = uint32(tmpID)
	}

	return farID, nil
}

func (upf *UPF) barID() (uint8, error) {
	if upf.UPFStatus != AssociatedSetUpSuccess {
		err := fmt.Errorf("this upf not associate with smf")
		return 0, err
	}

	var barID uint8
	if tmpID, err := upf.farIDGenerator.Allocate(); err != nil {
		return 0, err
	} else {
		barID = uint8(tmpID)
	}

	return barID, nil
}

func (upf *UPF) AddPDR() (*PDR, error) {
	if upf.UPFStatus != AssociatedSetUpSuccess {
		err := fmt.Errorf("this upf do not associate with smf")
		return nil, err
	}

	pdr := new(PDR)
	if PDRID, err := upf.pdrID(); err != nil {
		return nil, err
	} else {
		pdr.PDRID = PDRID
		upf.pdrPool.Store(pdr.PDRID, pdr)
	}

	if newFAR, err := upf.AddFAR(); err != nil {
		return nil, err
	} else {
		pdr.FAR = newFAR
	}

	return pdr, nil
}

func (upf *UPF) AddFAR() (*FAR, error) {

	if upf.UPFStatus != AssociatedSetUpSuccess {
		err := fmt.Errorf("this upf do not associate with smf")
		return nil, err
	}

	far := new(FAR)
	if FARID, err := upf.farID(); err != nil {
		return nil, err
	} else {
		far.FARID = FARID
		upf.farPool.Store(far.FARID, far)
	}

	return far, nil
}

func (upf *UPF) AddBAR() (*BAR, error) {

	if upf.UPFStatus != AssociatedSetUpSuccess {
		err := fmt.Errorf("this upf do not associate with smf")
		return nil, err
	}

	bar := new(BAR)
	if BARID, err := upf.barID(); err != nil {

	} else {
		bar.BARID = uint8(BARID)
		upf.barPool.Store(bar.BARID, bar)
	}

	return bar, nil
}

func (upf *UPF) RemovePDR(pdr *PDR) (err error) {

	if upf.UPFStatus != AssociatedSetUpSuccess {
		err = fmt.Errorf("this upf not associate with smf")
		return err
	}

	upf.pdrIDGenerator.FreeID(int64(pdr.PDRID))
	upf.pdrPool.Delete(pdr.PDRID)
	return nil
}

func (upf *UPF) RemoveFAR(far *FAR) (err error) {

	if upf.UPFStatus != AssociatedSetUpSuccess {
		err = fmt.Errorf("this upf not associate with smf")
		return err
	}

	upf.farIDGenerator.FreeID(int64(far.FARID))
	upf.farPool.Delete(far.FARID)
	return nil
}

func (upf *UPF) RemoveBAR(bar *BAR) (err error) {

	if upf.UPFStatus != AssociatedSetUpSuccess {
		err = fmt.Errorf("this upf not associate with smf")
		return err
	}

	upf.barIDGenerator.FreeID(int64(bar.BARID))
	upf.barPool.Delete(bar.BARID)
	return nil
}
