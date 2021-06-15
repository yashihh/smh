package oam

import (
	"github.com/gin-gonic/gin"

	"bitbucket.org/free5gc-team/http_wrapper"
	"bitbucket.org/free5gc-team/smf/internal/producer"
)

func HTTPGetUEPDUSessionInfo(c *gin.Context) {
	req := http_wrapper.NewRequest(c.Request, nil)
	req.Params["smContextRef"] = c.Params.ByName("smContextRef")

	smContextRef := req.Params["smContextRef"]
	HTTPResponse := producer.HandleOAMGetUEPDUSessionInfo(smContextRef)

	c.JSON(HTTPResponse.Status, HTTPResponse.Body)
}
