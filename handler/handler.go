package handler

import (
	"free5gc/src/smf/handler/message"
	"free5gc/src/smf/pfcp"
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
				}
			}
		case <-time.After(time.Second * 1):
		}
	}
}
