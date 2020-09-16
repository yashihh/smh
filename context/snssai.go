package context

type SNssai struct {
	Sst int32
	Sd  string
}

type SnssaiInfo struct {
	SNssai  SNssai
	DnnList []string
}
