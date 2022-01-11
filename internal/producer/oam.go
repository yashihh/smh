package producer

import (
	"net/http"
	"strconv"

	"bitbucket.org/free5gc-team/openapi/models"
	"bitbucket.org/free5gc-team/smf/internal/context"
	"bitbucket.org/free5gc-team/util/httpwrapper"
)

type PDUSessionInfo struct {
	Supi         string
	PDUSessionID string
	Dnn          string
	Sst          string
	Sd           string
	AnType       models.AccessType
	PDUAddress   string
	SessionRule  models.SessionRule
	UpCnxState   models.UpCnxState
	Tunnel       context.UPTunnel
}

func HandleOAMGetUEPDUSessionInfo(smContextRef string) *httpwrapper.Response {
	smContext := context.GetSMContextByRef(smContextRef)
	if smContext == nil {
		httpResponse := &httpwrapper.Response{
			Header: nil,
			Status: http.StatusNotFound,
			Body:   nil,
		}

		return httpResponse
	}

	httpResponse := &httpwrapper.Response{
		Header: nil,
		Status: http.StatusOK,
		Body: PDUSessionInfo{
			Supi:         smContext.Supi,
			PDUSessionID: strconv.Itoa(int(smContext.PDUSessionID)),
			Dnn:          smContext.Dnn,
			Sst:          strconv.Itoa(int(smContext.Snssai.Sst)),
			Sd:           smContext.Snssai.Sd,
			AnType:       smContext.AnType,
			PDUAddress:   smContext.PDUAddress.String(),
			UpCnxState:   smContext.UpCnxState,
			// Tunnel: context.UPTunnel{
			// 	//UpfRoot:  smContext.Tunnel.UpfRoot,
			// 	ULCLRoot: smContext.Tunnel.UpfRoot,
			// },
		},
	}
	return httpResponse
}
