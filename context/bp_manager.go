package context

import (
	"reflect"
)

type BPManager struct {
	BPStatus BPStatus

	//Need these variable conducting Add addtional PSA (TS23.502 4.3.5.4)
	//There value will change from time to time

	ActivatedPaths        []*DataPath
	ActivatingPath        *DataPath
	UpdatedBranchingPoint map[*UPF]int
	ULCL                  *UPF
}

type BPStatus int

const (
	UnInitialized BPStatus = iota
	HasSendPFCPMsg
	InitializedSuccess
	InitializedFail
)

type PDUSessionAnchorState int

const (
	NotAdded PDUSessionAnchorState = iota
	AddPSASuccess
	AddPSAFail
)

type ULCLState int

const (
	IsOnlyULCL ULCLState = iota
	IsULCLAndPSA1
	IsULCLAndPSA2
)

func NewBPManager(supi string) (bpManager *BPManager) {

	bpManager = &BPManager{
		BPStatus:              UnInitialized,
		ActivatedPaths:        make([]*DataPath, 0),
		UpdatedBranchingPoint: make(map[*UPF]int),
	}

	return
}

func (bpMGR *BPManager) SelectPSA2(smContext *SMContext) {

	hasSelectPSA2 := false
	bpMGR.ActivatedPaths = []*DataPath{}
	for _, dataPath := range smContext.Tunnel.DataPathPool {

		if dataPath.Activated {
			bpMGR.ActivatedPaths = append(bpMGR.ActivatedPaths, dataPath)
		} else {
			if !hasSelectPSA2 {
				bpMGR.ActivatingPath = dataPath
				hasSelectPSA2 = true
			}
		}
	}
}

func (bpMGR *BPManager) FindULCL(smContext *SMContext) (err error) {

	bpMGR.UpdatedBranchingPoint = make(map[*UPF]int)
	activatingPath := bpMGR.ActivatingPath
	for _, psa1Path := range bpMGR.ActivatedPaths {
		depth := 0
		psa1CurDPNode := psa1Path.FirstDPNode
		for psa2CurDPNode := activatingPath.FirstDPNode; psa2CurDPNode != nil; psa2CurDPNode = psa2CurDPNode.Next() {
			if reflect.DeepEqual(psa2CurDPNode.UPF.NodeID, psa1CurDPNode.UPF.NodeID) {

				psa1CurDPNode = psa1CurDPNode.Next()
				depth += 1

				if _, exist := bpMGR.UpdatedBranchingPoint[psa2CurDPNode.UPF]; !exist {
					bpMGR.UpdatedBranchingPoint[psa2CurDPNode.UPF] = depth
				}
			} else {
				break
			}

		}
	}

	maxDepth := 0
	for upf, depth := range bpMGR.UpdatedBranchingPoint {

		if depth > maxDepth {
			bpMGR.ULCL = upf
			maxDepth = depth
		}
	}
	return
}
