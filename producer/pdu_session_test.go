package producer_test

import (
	"bytes"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/h2non/gock.v1"

	"bitbucket.org/free5gc-team/http_wrapper"
	"bitbucket.org/free5gc-team/nas"
	"bitbucket.org/free5gc-team/nas/nasMessage"
	"bitbucket.org/free5gc-team/nas/nasType"
	"bitbucket.org/free5gc-team/openapi"
	"bitbucket.org/free5gc-team/openapi/Nsmf_PDUSession"
	"bitbucket.org/free5gc-team/openapi/models"
	"bitbucket.org/free5gc-team/smf/context"
	"bitbucket.org/free5gc-team/smf/factory"
	"bitbucket.org/free5gc-team/smf/producer"
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
	Info: &factory.Info{
		Version:     "1.0.0",
		Description: "SMF procdeure test configuration",
	},
	Configuration: &factory.Configuration{
		SmfName: "SMF Procedure Test",
		Sbi: &factory.Sbi{
			Scheme:       "http",
			RegisterIPv4: "127.0.0.1",
			BindingIPv4:  "127.0.0.1",
			Port:         8000,
		},
		PFCP: &factory.PFCP{
			Addr: "127.0.0.1",
		},
		NrfUri:               "http://127.0.0.10:8000",
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
					Sd:  "112232",
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
	openapi.InterceptH2CClient()
}

func initDiscUDMStubNRF() {
	searchResult := &models.SearchResult{
		ValidityPeriod: 100,
		NfInstances: []models.NfProfile{
			{
				NfInstanceId: "smf-unit-testing",
				NfType:       "UDM",
				NfStatus:     "REGISTERED",
				PlmnList: &[]models.PlmnId{
					{
						Mcc: "208",
						Mnc: "93",
					},
				},
				Ipv4Addresses: []string{
					"127.0.0.3",
				},
				NfServices: &[]models.NfService{
					{
						ServiceInstanceId: "0",
						ServiceName:       "nudm-sdm",
						Versions: &[]models.NfServiceVersion{
							{
								ApiVersionInUri: "v1",
								ApiFullVersion:  "1.0.0",
							},
						},
						Scheme:          "http",
						NfServiceStatus: "REGISTERED",
						IpEndPoints: &[]models.IpEndPoint{
							{
								Ipv4Address: "127.0.0.3",
								Transport:   "TCP",
								Port:        8000,
							},
						},
						ApiPrefix: "http://127.0.0.3:8000",
					},
				},
			},
		},
	}

	// intercept uri: http://127.0.0.10:8000/nnrf-disc/v1/nf-instances?requester-nf-type=SMF&target-nf-type=UDM
	gock.New("http://127.0.0.10:8000/nnrf-disc/v1").
		Get("/nf-instances").
		MatchParam("target-nf-type", "UDM").
		MatchParam("requester-nf-type", "SMF").
		Reply(http.StatusOK).
		JSON(searchResult)
}

func initDiscPCFStubNRF() {
	searchResult := &models.SearchResult{
		ValidityPeriod: 100,
		NfInstances: []models.NfProfile{
			{
				NfInstanceId: "smf-unit-testing",
				NfType:       "PCF",
				NfStatus:     "REGISTERED",
				PlmnList: &[]models.PlmnId{
					{
						Mcc: "208",
						Mnc: "93",
					},
				},
				Ipv4Addresses: []string{
					"127.0.0.7",
				},
				PcfInfo: &models.PcfInfo{
					DnnList: []string {
						"free5gc",
						"internet",
					},
				},
				NfServices: &[]models.NfService{
					{
						ServiceInstanceId: "1",
						ServiceName:       "npcf-smpolicycontrol",
						Versions: &[]models.NfServiceVersion{
							{
								ApiVersionInUri: "v1",
								ApiFullVersion:  "1.0.0",
							},
						},
						Scheme:          "http",
						NfServiceStatus: "REGISTERED",
						IpEndPoints: &[]models.IpEndPoint{
							{
								Ipv4Address: "127.0.0.7",
								Transport:   "TCP",
								Port:        8000,
							},
						},
						ApiPrefix: "http://127.0.0.7:8000",
					},
				},
			},
		},
	}

	// intercept uri: http://127.0.0.10:8000/nnrf-disc/v1/nf-instances?requester-nf-type=SMF&target-nf-type=PCF
	gock.New("http://127.0.0.10:8000/nnrf-disc/v1").
		Get("/nf-instances").
		MatchParam("target-nf-type", "PCF").
		MatchParam("requester-nf-type", "SMF").
		Reply(http.StatusOK).
		JSON(searchResult)

}

func initGetSMDataStubUDM() {
	SMSubscriptionData := []models.SessionManagementSubscriptionData{
		{
			SingleNssai: &models.Snssai{
				Sst: 1,
				Sd: "010203",
			}, 
			DnnConfigurations: map[string]models.DnnConfiguration{
				"internet": models.DnnConfiguration{
					PduSessionTypes: &models.PduSessionTypes{
						DefaultSessionType: "IPV4",
						AllowedSessionTypes: []models.PduSessionType{
							"IPV4",
						},
					},
					SscModes: &models.SscModes{
						DefaultSscMode: "SSC_MODE_1",
						AllowedSscModes: []models.SscMode{
							"SSC_MODE_1",
							"SSC_MODE_2",
							"SSC_MODE_3",
						},
					},
					Var5gQosProfile: &models.SubscribedDefaultQos{
						Var5qi: 9,
						Arp: &models.Arp{
							PriorityLevel: 8,
						},
						PriorityLevel: 8,
					},
					SessionAmbr: &models.Ambr{
						Uplink: "1000 Kbps", 
						Downlink: "1000 Kbps",
					},
				},
			},
		},
	}

	//intercept uri: http://127.0.0.3:8000/nudm-sdm/v1/imsi-2089300007487/sm-data?dnn=internet&plmn-id=20893&single-nssai=%7B%22sst%22%3A1%2C%22sd%22%3A%22010203%22%7D

	gock.New("http://127.0.0.3:8000/nudm-sdm/v1/imsi-2089300007487").
		Get("/sm-data").
		MatchParam("dnn", "internet").
		Reply(http.StatusOK).
		JSON(SMSubscriptionData)
}

func TestHandlePDUSessionSMContextCreate(t *testing.T) {
	//Prepare GSM Message
	GSMMsg := nasMessage.NewPDUSessionEstablishmentRequest(0)
	//Set GSM Message
	GSMMsg.PDUSessionID.SetPDUSessionID(10)
	GSMMsg.PTI.SetPTI(1)
	GSMMsg.PDUSessionType = nasType.NewPDUSessionType(nasMessage.PDUSessionEstablishmentRequestPDUSessionTypeType)
	GSMMsg.PDUSessionType.SetPDUSessionTypeValue(nasMessage.PDUSessionTypeIPv4)
	GSMMsg.PDUSESSIONESTABLISHMENTREQUESTMessageIdentity.SetMessageType(nas.MsgTypePDUSessionEstablishmentRequest)
	//Encode GSM Message
	buff := new(bytes.Buffer)
	GSMMsg.EncodePDUSessionEstablishmentRequest(buff)
	GSMMsgBytes := make([]byte, buff.Len())
	buff.Read(GSMMsgBytes)

	GSMMsgWrongType := nasMessage.NewPDUSessionModificationRequest(0)
	//Encode GSM Message
	buff = new(bytes.Buffer)
	GSMMsgWrongType.EncodePDUSessionModificationRequest(buff)
	GSMMsgWrongType.PDUSESSIONMODIFICATIONREQUESTMessageIdentity.SetMessageType(nas.MsgTypePDUSessionModificationRequest)
	GSMMsgWrongTypeBytes := make([]byte, buff.Len())
	buff.Read(GSMMsgWrongTypeBytes)

	initDiscUDMStubNRF()
	initGetSMDataStubUDM()
	initDiscPCFStubNRF()
	testCases := []struct {
		request              models.PostSmContextsRequest
		paramStr             string
		resultStr            string
		expectedHTTPResponse *http_wrapper.Response
	}{
		{
			request: models.PostSmContextsRequest{
				BinaryDataN1SmMessage: GSMMsgWrongTypeBytes,
			},
			paramStr:  "input wrong GSM Message type\n",
			resultStr: "PDUSessionSMContextCreate should fail due to wrong GSM type\n",
			expectedHTTPResponse: &http_wrapper.Response{
				Header: nil,
				Status: http.StatusForbidden,
				Body: models.PostSmContextsErrorResponse{
					JsonData: &models.SmContextCreateError{
						Error: &Nsmf_PDUSession.N1SmError,
					},
				},
			},
		}, {
			request: models.PostSmContextsRequest{
				JsonData: &models.SmContextCreateData{
					Supi:         "imsi-2089300007487",
					Pei:          "imeisv-1110000000000000",
					Gpsi:         "msisdn-0900000000",
					PduSessionId: 10,
					Dnn:          "internet",
					SNssai: &models.Snssai{
						Sst: 1,
						Sd:  "112232",
					},
					ServingNfId: "c8d0ee65-f466-48aa-a42f-235ec771cb52",
					Guami: &models.Guami{
						PlmnId: &models.PlmnId{
							Mcc: "208",
							Mnc: "93",
						},
						AmfId: "cafe00",
					},
					AnType: "3GPP_ACCESS",
				},
				BinaryDataN1SmMessage: GSMMsgBytes,
			},
			paramStr:  "input wrong GSM Message type\n",
			resultStr: "PDUSessionSMContextCreate should fail due to wrong GSM type\n",
			expectedHTTPResponse: &http_wrapper.Response{
				Header: nil,
				Status: http.StatusForbidden,
				Body: models.PostSmContextsErrorResponse{
					JsonData: &models.SmContextCreateError{
						Error: &Nsmf_PDUSession.N1SmError,
					},
				},
			},
		},
	}

	//Find UDM
	//
	Convey("Procdure Test: Handle PDUSession SMContext Create", t, func() {
		for i, testcase := range testCases {
			infoStr := fmt.Sprintf("testcase[%d]: ", i)

			Convey(infoStr, func() {
				Convey(testcase.paramStr, func() {
					httpResp := producer.HandlePDUSessionSMContextCreate(testcase.request)

					Convey(testcase.resultStr, func() {
						So(true, ShouldEqual, reflect.DeepEqual(httpResp, testcase.expectedHTTPResponse))
					})
				})
			})
		}
	})
}

