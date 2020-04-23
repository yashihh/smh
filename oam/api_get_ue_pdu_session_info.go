package oam

import (
	"free5gc/lib/http_wrapper"
	"free5gc/src/smf/handler/smf_message"
	"github.com/gin-gonic/gin"
)

func GetUEPDUSessionInfo(c *gin.Context) {

	req := http_wrapper.NewRequest(c.Request, nil)
	req.Params["smContextRef"] = c.Params.ByName("smContextRef")

	handlerMsg := smf_message.NewHandlerMessage(smf_message.OAMGetUEPDUSessionInfo, req)
	smf_message.SendMessage(handlerMsg)

	rsp := <-handlerMsg.ResponseChan

	HTTPResponse := rsp.HTTPResponse

	c.JSON(HTTPResponse.Status, HTTPResponse.Body)
}
