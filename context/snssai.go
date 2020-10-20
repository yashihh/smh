package context

import "bitbucket.org/free5gc-team/openapi/models"

type SNssai struct {
	Sst int32
	Sd  string
}

type SnssaiUPFInfo struct {
	SNssai  SNssai
	DnnList []DnnUPFInfoItem
}

// DnnUpfInfoItem presents UPF dnn information
type DnnUPFInfoItem struct {
	Dnn             string
	DnaiList        []string
	PduSessionTypes []models.PduSessionType
}

// ContainsDNAI return true if the this dnn Info contains the specify DNAI
func (d *DnnUPFInfoItem) ContainsDNAI(targetDnai string) bool {
	if targetDnai == "" {
		return d.DnaiList == nil || len(d.DnaiList) == 0
	}
	for _, dnai := range d.DnaiList {
		if dnai == targetDnai {
			return true
		}
	}
	return false
}
