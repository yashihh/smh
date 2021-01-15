package producer_test

import (
	"bitbucket.org/free5gc-team/smf/factory"
	"bitbucket.org/free5gc-team/smf/context"
	//"testing"
	//"bitbucket.org/free5gc-team/http_wrapper"
	"bitbucket.org/free5gc-team/openapi/models"
)
var userPlaneConfig = factory.UserPlaneInformation{
	UPNodes: map[string]factory.UPNode{
		"GNodeB": {
			Type:   "AN",
			NodeID: "192.168.179.100",
		},
		"UPF1": {
			Type:   "UPF",
			NodeID: "192.168.179.1",
			SNssaiInfos: []models.SnssaiUpfInfoItem{
				{
					SNssai: &models.Snssai{
						Sst: 1,
						Sd:  "112232",
					},
					DnnUpfInfoList: []models.DnnUpfInfoItem{
						{Dnn: "internet"},
					},
				},
			},
		},
	},
	Links: []factory.UPLink{
		{
			A: "GNodeB",
			B: "UPF1",
		},
	},
}
var testConfig = factory.Config{
	Info: &factory.Info {
		Version: "1.0.0",
		Description: "SMF procdeure test configuration",
	},
	Configuration: &factory.Configuration{
		SmfName: "SMF Procedure Test",
		Sbi: &factory.Sbi{
			Scheme: "http",
			RegisterIPv4: "127.0.0.1",
			BindingIPv4: "127.0.0.1",
			Port: 8000,
		},
		PFCP: &factory.PFCP{
			Addr: "127.0.0.1",
		},
		NrfUri: "http://127.0.0.10:8000",
		UserPlaneInformation: userPlaneConfig,
		ServiceNameList: []string{
			"nsmf-pdusession",
			"nsmf-event-exposure",
			"nsmf-oam",
		},
		SNssaiInfo: []factory.SnssaiInfoItem{
			{
				SNssai: &models.Snssai{
					Sst: 1,
					Sd: "112232",
				}, 
				DnnInfos: []factory.SnssaiDnnInfoItem{
					{	
						Dnn: "internet",
						DNS: factory.DNS{
							IPv4Addr: "8.8.8.8",
							IPv6Addr: "2001:4860:4860::8888",
						},
						UESubnet: "60.60.0.0/16",
					},
				},
			},
		},
	},
}

func init() {
	context.InitSmfContext(&testConfig)
}

// func TestHandlePDUSessionSMContextCreate(t *testing.T) {

// 	InitSmfContext()
// 	testCases := []struct {
// 		Request models.PostSmContextsRequest
// 		paramStr              string
// 		resultStr             string
// 		expectedHTTPResponse  *http_wrapper.Response
// 	}{{
// 		Request: models.PostSmContextsRequest{
// 			JsonData: &models.SmContextCreateData{
// 				Supi:

// 			},
// 		},

// 	},}
// }
