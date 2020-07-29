package message

type Event int

const (
	PFCPMessage Event = iota
	PDUSessionSMContextUpdate
	PDUSessionSMContextRelease
)
