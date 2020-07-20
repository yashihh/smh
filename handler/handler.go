package handler

import (
	"free5gc/lib/openapi/models"
	"free5gc/src/smf/handler/message"
	"free5gc/src/smf/pfcp"
	"free5gc/src/smf/producer"
	"time"
)

func Handle() {

	for {
		select {
		case msg, ok := <-message.SmfChannel:
			if ok {
				switch msg.Event {
				case message.PFCPMessage:
					pfcp.Dispatch(msg.PFCPRequest)
				case message.OAMGetUEPDUSessionInfo:
					smContextRef := msg.HTTPRequest.Params["smContextRef"]
					producer.HandleOAMGetUEPDUSessionInfo(msg.ResponseChan, smContextRef)
				case message.SMPolicyUpdateNotify:
					smContextRef := msg.HTTPRequest.Params["smContextRef"]
					producer.HandleSMPolicyUpdateNotify(msg.ResponseChan, smContextRef, msg.HTTPRequest.Body.(models.SmPolicyNotification))
				}
			}
		case <-time.After(time.Second * 1):
		}
	}
}
