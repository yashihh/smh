package context_test

import (
	"fmt"
	"net"
	"testing"

	"bitbucket.org/free5gc-team/nas/nasMessage"
	"bitbucket.org/free5gc-team/smf/context"
	. "github.com/smartystreets/goconvey/convey"
)

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
		expectedExist bool
	}{
		{
			context.NewUPTunnel(),
			context.NewDataPath(),
			true,
		},
	}

	Convey("", t, func() {

		for i, testcase := range testCases {
			upTunnel := testcase.tunnel
			infoStr := fmt.Sprintf("testcase[%d]: Add Datapath", i)
			Convey(infoStr, func() {
				upTunnel.AddDataPath(testcase.addedDataPath)

				var resultStr string
				if testcase.expectedExist {
					resultStr = "Datapath should exist"
				} else {
					resultStr = "Datapath should not exist"
				}

				Convey(resultStr, func() {
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
