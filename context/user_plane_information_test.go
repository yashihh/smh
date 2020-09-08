package context_test

import (
	"free5gc/lib/openapi/models"
	"free5gc/src/smf/context"
	"free5gc/src/smf/factory"
	"testing"

	"github.com/stretchr/testify/require"
)

var configuration = &factory.UserPlaneInformation{
	UPNodes: map[string]factory.UPNode{
		"GNodeB": {
			Type:   "AN",
			NodeID: "192.168.179.100",
		},
		"UPF1": {
			Type:   "UPF",
			NodeID: "192.168.179.1",
			SNssaiInfo: models.SnssaiSmfInfoItem{
				SNssai: &models.Snssai{
					Sst: 1,
					Sd:  "112232",
				},
				DnnSmfInfoList: &[]models.DnnSmfInfoItem{},
			},
		},
		"UPF2": {
			Type:   "UPF",
			NodeID: "192.168.179.2",
			SNssaiInfo: models.SnssaiSmfInfoItem{
				SNssai: &models.Snssai{
					Sst: 2,
					Sd:  "112233",
				},
				DnnSmfInfoList: &[]models.DnnSmfInfoItem{},
			},
		},
		"UPF3": {
			Type:   "UPF",
			NodeID: "192.168.179.3",
			SNssaiInfo: models.SnssaiSmfInfoItem{
				SNssai: &models.Snssai{
					Sst: 3,
					Sd:  "112234",
				},
				DnnSmfInfoList: &[]models.DnnSmfInfoItem{},
			},
		},
		"UPF4": {
			Type:   "UPF",
			NodeID: "192.168.179.4",
			SNssaiInfo: models.SnssaiSmfInfoItem{
				SNssai: &models.Snssai{
					Sst: 1,
					Sd:  "112235",
				},
				DnnSmfInfoList: &[]models.DnnSmfInfoItem{},
			},
		},
	},
	Links: []factory.UPLink{
		{
			A: "GNodeB",
			B: "UPF1",
		},
		{
			A: "UPF1",
			B: "UPF2",
		},
		{
			A: "UPF2",
			B: "UPF3",
		},
		{
			A: "UPF3",
			B: "UPF4",
		},
	},
}

func TestNewUserPlaneInformation(t *testing.T) {
	userplaneInformation := context.NewUserPlaneInformation(configuration)

	require.NotNil(t, userplaneInformation.AccessNetwork["GNodeB"])

	require.NotNil(t, userplaneInformation.UPFs["UPF1"])
	require.NotNil(t, userplaneInformation.UPFs["UPF2"])
	require.NotNil(t, userplaneInformation.UPFs["UPF3"])
	require.NotNil(t, userplaneInformation.UPFs["UPF4"])

	// check links
	require.Contains(t, userplaneInformation.AccessNetwork["GNodeB"].Links, userplaneInformation.UPFs["UPF1"])
	require.Contains(t, userplaneInformation.UPFs["UPF1"].Links, userplaneInformation.UPFs["UPF2"])
	require.Contains(t, userplaneInformation.UPFs["UPF2"].Links, userplaneInformation.UPFs["UPF3"])
	require.Contains(t, userplaneInformation.UPFs["UPF3"].Links, userplaneInformation.UPFs["UPF4"])

}

func TestGenerateDefaultPath(t *testing.T) {
	configuration.Links = []factory.UPLink{
		{
			A: "GNodeB",
			B: "UPF1",
		},
		{
			A: "GNodeB",
			B: "UPF2",
		},
		{
			A: "GNodeB",
			B: "UPF3",
		},
		{
			A: "UPF1",
			B: "UPF4",
		},
	}

	upfSelectionParams := &context.UPFSelectionParams{
		SNssai: &context.SNssai{
			Sst: 1,
			Sd:  "112235",
		},
		Dnn: "internet",
	}

	userplaneInformation := context.NewUserPlaneInformation(configuration)
	userplaneInformation.UPFs["UPF1"].UPF.UPIPInfo.NetworkInstance = []uint8("internet")
	userplaneInformation.UPFs["UPF2"].UPF.UPIPInfo.NetworkInstance = []uint8("internet")
	userplaneInformation.UPFs["UPF3"].UPF.UPIPInfo.NetworkInstance = []uint8("internet")
	userplaneInformation.UPFs["UPF4"].UPF.UPIPInfo.NetworkInstance = []uint8("internet")

	//testcase 1
	pathExist := userplaneInformation.GenerateDefaultPath(upfSelectionParams)
	require.Equal(t, pathExist, true)

	//testcase 2
	upfSelectionParams.SNssai.Sst = 2
	pathExist = userplaneInformation.GenerateDefaultPath(upfSelectionParams)
	require.Equal(t, pathExist, true)

	//testcase 3
	upfSelectionParams.SNssai.Sst = 3
	pathExist = userplaneInformation.GenerateDefaultPath(upfSelectionParams)
	require.Equal(t, pathExist, true)

	//testcase 4
	upfSelectionParams.SNssai.Sst = 4
	pathExist = userplaneInformation.GenerateDefaultPath(upfSelectionParams)
	require.Equal(t, pathExist, false)

	//testcase 5
	upfSelectionParams.SNssai.Sst = 1
	configuration.UPNodes["UPF1"].SNssaiInfo.SNssai.Sst = 2
	userplaneInformation = context.NewUserPlaneInformation(configuration)
	userplaneInformation.UPFs["UPF1"].UPF.UPIPInfo.NetworkInstance = []uint8("internet")
	userplaneInformation.UPFs["UPF2"].UPF.UPIPInfo.NetworkInstance = []uint8("internet")
	userplaneInformation.UPFs["UPF3"].UPF.UPIPInfo.NetworkInstance = []uint8("internet")
	userplaneInformation.UPFs["UPF4"].UPF.UPIPInfo.NetworkInstance = []uint8("internet")
	pathExist = userplaneInformation.GenerateDefaultPath(upfSelectionParams)
	require.Equal(t, pathExist, false)
}

func TestGetDefaultUPFTopoByDNN(t *testing.T) {
}
