package context_test

import (
	"fmt"
	"net"
	"testing"

	"bitbucket.org/free5gc-team/nas/nasMessage"
	"bitbucket.org/free5gc-team/pfcp/pfcpType"
	"bitbucket.org/free5gc-team/smf/context"
	"bitbucket.org/free5gc-team/smf/factory"
	. "github.com/smartystreets/goconvey/convey"
)

var mockIPv4NodeID = &pfcpType.NodeID{
	NodeIdType:  pfcpType.NodeIdTypeIpv4Address,
	NodeIdValue: net.ParseIP("127.0.0.1"),
}

var mockIfaces = []factory.InterfaceUpfInfoItem{
	{
		InterfaceType:   "N3",
		Endpoints:       []string{"127.0.0.1"},
		NetworkInstance: "internet",
	},
}

func convertPDUSessTypeToString(PDUtype uint8) string {
	switch PDUtype {
	case nasMessage.PDUSessionTypeIPv4:
		return "PDU Session Type IPv4"
	case nasMessage.PDUSessionTypeIPv6:
		return "PDU Session Type IPv6"
	case nasMessage.PDUSessionTypeIPv4IPv6:
		return "PDU Session Type IPv4 IPv6"
	case nasMessage.PDUSessionTypeUnstructured:
		return "PDU Session Type Unstructured"
	case nasMessage.PDUSessionTypeEthernet:
		return "PDU Session Type Ethernet"
	}

	return "Unkwown PDU Session Type"
}

func TestIP(t *testing.T) {

	var testCases = []struct {
		input               *context.UPFInterfaceInfo
		inputPDUSessionType uint8
		expectedIP          string
		expectedError       error
	}{
		{
			&context.UPFInterfaceInfo{
				NetworkInstance:       "",
				IPv4EndPointAddresses: []net.IP{net.ParseIP("8.8.8.8")},
				IPv6EndPointAddresses: []net.IP{net.ParseIP("2001:4860:4860::8888")},
				EndpointFQDN:          "www.google.com",
			},
			nasMessage.PDUSessionTypeIPv4,
			"8.8.8.8",
			nil,
		},
		{
			&context.UPFInterfaceInfo{
				NetworkInstance:       "",
				IPv4EndPointAddresses: []net.IP{net.ParseIP("8.8.8.8")},
				IPv6EndPointAddresses: []net.IP{net.ParseIP("2001:4860:4860::8888")},
				EndpointFQDN:          "www.google.com",
			},
			nasMessage.PDUSessionTypeIPv6,
			"2001:4860:4860::8888",
			nil,
		},
	}

	Convey("", t, func() {
		for i, testcase := range testCases {

			upfInterfaceInfo := testcase.input
			infoStr := fmt.Sprintf("testcase[%d] UPF Interface Info: %+v", i, upfInterfaceInfo)

			Convey(infoStr, func() {
				Convey("select "+convertPDUSessTypeToString(testcase.inputPDUSessionType), func() {
					ip, err := upfInterfaceInfo.IP(testcase.inputPDUSessionType)
					Convey("IP addr should be "+testcase.expectedIP, func() {

						So(ip.String(), ShouldEqual, testcase.expectedIP)
						So(err, ShouldEqual, testcase.expectedError)
					})
				})
			})
		}
	})
}

func TestAddDataPath(t *testing.T) {
	//AddDataPath is simple, should only have one case
	var testCases = []struct {
		tunnel        *context.UPTunnel
		addedDataPath *context.DataPath
		resultStr     string
		expectedExist bool
	}{
		{
			context.NewUPTunnel(),
			context.NewDataPath(),
			"Datapath should exist",
			true,
		},
	}

	Convey("", t, func() {

		for i, testcase := range testCases {
			upTunnel := testcase.tunnel
			infoStr := fmt.Sprintf("testcase[%d]: Add Datapath", i)
			Convey(infoStr, func() {
				upTunnel.AddDataPath(testcase.addedDataPath)

				Convey(testcase.resultStr, func() {
					var exist bool
					for _, datapath := range upTunnel.DataPathPool {

						if datapath == testcase.addedDataPath {
							exist = true
						}
					}
					So(exist, ShouldEqual, testcase.expectedExist)
				})
			})
		}
	})

}

func TestAddPDR(t *testing.T) {
	var testCases = []struct {
		upf           *context.UPF
		resultStr     string
		expectedError error
	}{
		{
			context.NewUPF(mockIPv4NodeID, mockIfaces),
			"AddPDR should success",
			nil,
		},
		{
			context.NewUPF(mockIPv4NodeID, mockIfaces),
			"AddPDR should fail",
			fmt.Errorf("this upf do not associate with smf"),
		},
	}

	testCases[0].upf.UPFStatus = context.AssociatedSetUpSuccess

	Convey("", t, func() {

		for i, testcase := range testCases {
			upf := testcase.upf
			infoStr := fmt.Sprintf("testcase[%d]: ", i)
			Convey(infoStr, func() {
				_, err := upf.AddPDR()

				Convey(testcase.resultStr, func() {
					if testcase.expectedError == nil {
						So(err, ShouldBeNil)
					} else {
						So(err, ShouldNotBeNil)
						if err != nil {
							So(err.Error(), ShouldEqual, testcase.expectedError.Error())
						}
					}

				})
			})
		}
	})
}

func TestAddFAR(t *testing.T) {
	var testCases = []struct {
		upf           *context.UPF
		resultStr     string
		expectedError error
	}{
		{
			context.NewUPF(mockIPv4NodeID, mockIfaces),
			"AddFAR should success",
			nil,
		},
		{
			context.NewUPF(mockIPv4NodeID, mockIfaces),
			"AddFAR should fail",
			fmt.Errorf("this upf do not associate with smf"),
		},
	}

	testCases[0].upf.UPFStatus = context.AssociatedSetUpSuccess

	Convey("", t, func() {

		for i, testcase := range testCases {
			upf := testcase.upf
			infoStr := fmt.Sprintf("testcase[%d]: ", i)
			Convey(infoStr, func() {
				_, err := upf.AddFAR()

				Convey(testcase.resultStr, func() {
					if testcase.expectedError == nil {
						So(err, ShouldBeNil)
					} else {
						So(err, ShouldNotBeNil)
						if err != nil {
							So(err.Error(), ShouldEqual, testcase.expectedError.Error())
						}
					}

				})
			})
		}
	})
}

func TestAddQER(t *testing.T) {
	var testCases = []struct {
		upf           *context.UPF
		resultStr     string
		expectedError error
	}{
		{
			context.NewUPF(mockIPv4NodeID, mockIfaces),
			"AddQER should success",
			nil,
		},
		{
			context.NewUPF(mockIPv4NodeID, mockIfaces),
			"AddQER should fail",
			fmt.Errorf("this upf do not associate with smf"),
		},
	}

	testCases[0].upf.UPFStatus = context.AssociatedSetUpSuccess

	Convey("", t, func() {

		for i, testcase := range testCases {
			upf := testcase.upf
			infoStr := fmt.Sprintf("testcase[%d]: ", i)
			Convey(infoStr, func() {
				_, err := upf.AddQER()

				Convey(testcase.resultStr, func() {
					if testcase.expectedError == nil {
						So(err, ShouldBeNil)
					} else {
						So(err, ShouldNotBeNil)
						if err != nil {
							So(err.Error(), ShouldEqual, testcase.expectedError.Error())
						}
					}

				})
			})
		}
	})
}

func TestAddBAR(t *testing.T) {
	var testCases = []struct {
		upf           *context.UPF
		resultStr     string
		expectedError error
	}{
		{
			context.NewUPF(mockIPv4NodeID, mockIfaces),
			"AddBAR should success",
			nil,
		},
		{
			context.NewUPF(mockIPv4NodeID, mockIfaces),
			"AddBAR should fail",
			fmt.Errorf("this upf do not associate with smf"),
		},
	}

	testCases[0].upf.UPFStatus = context.AssociatedSetUpSuccess

	Convey("", t, func() {

		for i, testcase := range testCases {
			upf := testcase.upf
			infoStr := fmt.Sprintf("testcase[%d]: ", i)
			Convey(infoStr, func() {
				_, err := upf.AddBAR()

				Convey(testcase.resultStr, func() {
					if testcase.expectedError == nil {
						So(err, ShouldBeNil)
					} else {
						So(err, ShouldNotBeNil)
						if err != nil {
							So(err.Error(), ShouldEqual, testcase.expectedError.Error())
						}
					}

				})
			})
		}
	})
}
