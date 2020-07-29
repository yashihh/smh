package context

import (
	"fmt"
	"free5gc/lib/idgenerator"
	"free5gc/lib/pfcp/pfcpType"
	"free5gc/lib/pfcp/pfcpUdp"
	"free5gc/src/smf/logger"
	"github.com/google/uuid"
	"math"
	"net"
	"reflect"
	"sync"
)

var upfPool map[string]*UPF

func init() {
	upfPool = make(map[string]*UPF)
}

type UPTunnel struct {
	PathIDGenerator *idgenerator.IDGenerator
	DataPathPool    DataPathPool
}

type UPFStatus int

const (
	NotAssociated          UPFStatus = 0
	AssociatedSettingUp    UPFStatus = 1
	AssociatedSetUpSuccess UPFStatus = 2
)

type UPF struct {
	uuid      uuid.UUID
	NodeID    pfcpType.NodeID
	UPIPInfo  pfcpType.UserPlaneIPResourceInformation
	UPFStatus UPFStatus

	pdrPool        sync.Map
	farPool        sync.Map
	barPool        sync.Map
	urrPool        sync.Map
	qerPool        sync.Map
	pdrIDGenerator *idgenerator.IDGenerator
	farIDGenerator *idgenerator.IDGenerator
	barIDGenerator *idgenerator.IDGenerator
	urrIDGenerator *idgenerator.IDGenerator
	qerIDGenerator *idgenerator.IDGenerator
	teidGenerator  *idgenerator.IDGenerator
}

// UUID return this UPF UUID (allocate by SMF in this time)
// Maybe allocate by UPF in future
func (upf *UPF) UUID() string {
	return upf.uuid.String()
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
func NewUPF(nodeID *pfcpType.NodeID) (upf *UPF) {
	upf = new(UPF)
	upf.uuid = uuid.New()

	upfPool[upf.UUID()] = upf

	// Initialize context
	upf.UPFStatus = NotAssociated
	upf.NodeID = *nodeID
	upf.pdrIDGenerator = idgenerator.NewGenerator(1, math.MaxUint16)
	upf.farIDGenerator = idgenerator.NewGenerator(1, math.MaxUint32)
	upf.barIDGenerator = idgenerator.NewGenerator(1, math.MaxUint8)
	upf.qerIDGenerator = idgenerator.NewGenerator(1, math.MaxUint32)
	upf.urrIDGenerator = idgenerator.NewGenerator(1, math.MaxUint32)
	upf.teidGenerator = idgenerator.NewGenerator(1, math.MaxUint32)

	return upf
}

func (upf *UPF) GenerateTEID() (id uint32, err error) {
	if upf.UPFStatus != AssociatedSetUpSuccess {
		err := fmt.Errorf("this upf not associate with smf")
		return 0, err
	}

	var tmpID int64
	tmpID, err = upf.teidGenerator.Allocate()
	id = uint32(tmpID)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (upf *UPF) PFCPAddr() *net.UDPAddr {
	return &net.UDPAddr{
		IP:   upf.NodeID.ResolveNodeIdToIp(),
		Port: pfcpUdp.PFCP_PORT,
	}
}

func RetrieveUPFNodeByNodeID(nodeID pfcpType.NodeID) (upf *UPF) {
	for _, upf := range upfPool {
		if reflect.DeepEqual(upf.NodeID, nodeID) {
			return upf
		}
	}
	return nil
}

func RemoveUPFNodeByNodeId(nodeID pfcpType.NodeID) {
	for upfID, upf := range upfPool {
		if reflect.DeepEqual(upf.NodeID, nodeID) {
			delete(upfPool, upfID)
			break
		}
	}
}

func SelectUPFByDnn(Dnn string) *UPF {
	for _, upf := range upfPool {
		if upf.UPIPInfo.Assoni && string(upf.UPIPInfo.NetworkInstance) == Dnn {
			return upf
		}
	}
	return nil
}

func (upf *UPF) GetUPFIP() string {
	return upf.NodeID.ResolveNodeIdToIp().String()
}

func (upf *UPF) GetUPFID() string {

	upInfo := GetUserPlaneInformation()
	upfIP := upf.NodeID.ResolveNodeIdToIp().String()
	return upInfo.GetUPFIDByIP(upfIP)

}

func (upf *UPF) pdrID() (pdrID uint16, err error) {
	if upf.UPFStatus != AssociatedSetUpSuccess {
		err := fmt.Errorf("this upf not associate with smf")
		return 0, err
	}

	var tmpID int64
	tmpID, err = upf.pdrIDGenerator.Allocate()
	pdrID = uint16(tmpID)
	if err != nil {
		return 0, err
	}

	return pdrID, nil
}

func (upf *UPF) farID() (farID uint32, err error) {
	if upf.UPFStatus != AssociatedSetUpSuccess {
		err := fmt.Errorf("this upf not associate with smf")
		return 0, err
	}

	var tmpID int64
	tmpID, err = upf.farIDGenerator.Allocate()
	farID = uint32(tmpID)
	if err != nil {
		return 0, err
	}

	return farID, nil
}

func (upf *UPF) barID() (barID uint8, err error) {
	if upf.UPFStatus != AssociatedSetUpSuccess {
		err := fmt.Errorf("this upf not associate with smf")
		return 0, err
	}

	var tmpID int64
	tmpID, err = upf.barIDGenerator.Allocate()
	barID = uint8(tmpID)
	if err != nil {
		return 0, err
	}

	return barID, nil
}

func (upf *UPF) AddPDR() (pdr *PDR, err error) {

	if upf.UPFStatus != AssociatedSetUpSuccess {
		err = fmt.Errorf("this upf do not associate with smf")
		return nil, err
	}

	pdr = new(PDR)
	PDRID, _ := upf.pdrID()
	pdr.PDRID = PDRID
	upf.pdrPool.Store(pdr.PDRID, pdr)
	pdr.FAR, _ = upf.AddFAR()
	return pdr, nil
}

func (upf *UPF) AddFAR() (far *FAR, err error) {

	if upf.UPFStatus != AssociatedSetUpSuccess {
		err = fmt.Errorf("this upf do not associate with smf")
		return nil, err
	}

	far = new(FAR)
	far.FARID, _ = upf.farID()
	upf.farPool.Store(far.FARID, far)
	return far, nil
}

func (upf *UPF) AddBAR() (bar *BAR, err error) {

	if upf.UPFStatus != AssociatedSetUpSuccess {
		err = fmt.Errorf("this upf do not associate with smf")
		return nil, err
	}

	bar = new(BAR)
	BARID, _ := upf.barID()
	bar.BARID = uint8(BARID)
	upf.barPool.Store(bar.BARID, bar)
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
