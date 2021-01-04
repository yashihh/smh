package context_test

// import (
// 	"fmt"
// 	"net"
// 	"testing"

// 	"bitbucket.org/free5gc-team/nas/nasMessage"
// 	"bitbucket.org/free5gc-team/smf/context"
// 	. "github.com/smartystreets/goconvey/convey"
// )

// func TestIP(t *testing.T) {

// 	var testCases = []struct {
// 		input               *context.UPFInterfaceInfo
// 		inputPDUSessionType uint8
// 		expectedIP          string
// 		expectedError       error
// 	}{
// 		{
// 			&context.UPFInterfaceInfo{
// 				NetworkInstance:       "",
// 				IPv4EndPointAddresses: []net.IP{net.ParseIP("8.8.8.8")},
// 				IPv6EndPointAddresses: []net.IP{net.ParseIP("2001:4860:4860::8888")},
// 				EndpointFQDN:          "www.google.com",
// 			},
// 			nasMessage.PDUSessionTypeIPv4,
// 			"8.8.8.8",
// 			nil,
// 		},
// 		{
// 			&context.UPFInterfaceInfo{
// 				NetworkInstance:       "",
// 				IPv4EndPointAddresses: []net.IP{},
// 				IPv6EndPointAddresses: []net.IP{},
// 				EndpointFQDN:          "www.google.com",
// 			},
// 			nasMessage.PDUSessionTypeIPv4,
// 			"8.8.8.8",
// 			nil,
// 		},
// 	}

// 	Convey("Given different upf interface info structrues as follows", t, func() {

// 		for i, testcase := range testCases {

// 			upfInterfaceInfo := testcase.input
// 			infoStr := fmt.Sprintf("tc[%d] UPF Interface Info: %+v", i, upfInterfaceInfo)

// 			Convey(infoStr, func() {
// 				Convey("select IPv4 pdusession type", func() {
// 					ip, err := upfInterfaceInfo.input.IP(testcase.inputPDUSessionType)
// 					Convey("IP addr should be "+testcase.expectedIP, func() {
// 						So(testcase.expectedIP, ShouldEqual, ip.String())
// 					})

// 				})

// 				Convey("select IPv6 pdusession type", func() {

// 					Convey("IP addr should be", nil)

// 				})
// 			})

// 		}
// 	})

// }
