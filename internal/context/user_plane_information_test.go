package context

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"bitbucket.org/free5gc-team/openapi/models"
	"bitbucket.org/free5gc-team/smf/pkg/factory"
)

var configuration = &factory.UserPlaneInformation{
	UPNodes: map[string]*factory.UPNode{
		"GNodeB": {
			Type:   "AN",
			NodeID: "192.168.179.100",
		},
		"UPF1": {
			Type:   "UPF",
			NodeID: "192.168.179.1",
			SNssaiInfos: []*factory.SnssaiUpfInfoItem{
				{
					SNssai: &models.Snssai{
						Sst: 1,
						Sd:  "112232",
					},
					DnnUpfInfoList: []*factory.DnnUpfInfoItem{
						{
							Dnn: "internet",
							Pools: []*factory.UEIPPool{
								{
									Cidr: "10.60.0.0/27",
								},
							},
							StaticPools: []*factory.UEIPPool{
								{
									Cidr: "10.60.0.0/28",
								},
							},
						},
					},
				},
				{
					SNssai: &models.Snssai{
						Sst: 1,
						Sd:  "112235",
					},
					DnnUpfInfoList: []*factory.DnnUpfInfoItem{
						{
							Dnn: "internet",
							Pools: []*factory.UEIPPool{
								{
									Cidr: "10.61.0.0/16",
								},
							},
						},
					},
				},
			},
		},
		"UPF2": {
			Type:   "UPF",
			NodeID: "192.168.179.2",
			SNssaiInfos: []*factory.SnssaiUpfInfoItem{
				{
					SNssai: &models.Snssai{
						Sst: 2,
						Sd:  "112233",
					},
					DnnUpfInfoList: []*factory.DnnUpfInfoItem{
						{
							Dnn: "internet",
							Pools: []*factory.UEIPPool{
								{
									Cidr: "10.62.0.0/16",
								},
							},
						},
					},
				},
			},
		},
		"UPF3": {
			Type:   "UPF",
			NodeID: "192.168.179.3",
			SNssaiInfos: []*factory.SnssaiUpfInfoItem{
				{
					SNssai: &models.Snssai{
						Sst: 3,
						Sd:  "112234",
					},
					DnnUpfInfoList: []*factory.DnnUpfInfoItem{
						{
							Dnn: "internet",
							Pools: []*factory.UEIPPool{
								{
									Cidr: "10.63.0.0/16",
								},
							},
						},
					},
				},
			},
		},
		"UPF4": {
			Type:   "UPF",
			NodeID: "192.168.179.4",
			SNssaiInfos: []*factory.SnssaiUpfInfoItem{
				{
					SNssai: &models.Snssai{
						Sst: 1,
						Sd:  "112235",
					},
					DnnUpfInfoList: []*factory.DnnUpfInfoItem{
						{
							Dnn: "internet",
							Pools: []*factory.UEIPPool{
								{
									Cidr: "10.64.0.0/16",
								},
							},
						},
					},
				},
			},
		},
	},
	Links: []*factory.UPLink{
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
	userplaneInformation := NewUserPlaneInformation(configuration)

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
	configuration.Links = []*factory.UPLink{
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

	testCases := []struct {
		name     string
		param    *UPFSelectionParams
		expected bool
	}{
		{
			"S-NSSAI 01112232 and DNN internet ok",
			&UPFSelectionParams{
				SNssai: &SNssai{
					Sst: 1,
					Sd:  "112232",
				},
				Dnn: "internet",
			},
			true,
		},
		{
			"S-NSSAI 02112233 and DNN internet ok",
			&UPFSelectionParams{
				SNssai: &SNssai{
					Sst: 2,
					Sd:  "112233",
				},
				Dnn: "internet",
			},
			true,
		},
		{
			"S-NSSAI 03112234 and DNN internet ok",
			&UPFSelectionParams{
				SNssai: &SNssai{
					Sst: 3,
					Sd:  "112234",
				},
				Dnn: "internet",
			},
			true,
		},
		{
			"S-NSSAI 01112235 and DNN internet ok",
			&UPFSelectionParams{
				SNssai: &SNssai{
					Sst: 1,
					Sd:  "112235",
				},
				Dnn: "internet",
			},
			true,
		},
		{
			"S-NSSAI 01010203 and DNN internet fail",
			&UPFSelectionParams{
				SNssai: &SNssai{
					Sst: 1,
					Sd:  "010203",
				},
				Dnn: "internet",
			},
			false,
		},
	}

	userplaneInformation := NewUserPlaneInformation(configuration)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pathExist := userplaneInformation.GenerateDefaultPath(tc.param)
			require.Equal(t, tc.expected, pathExist)
		})
	}
}

func TestGetDefaultUPFTopoByDNN(t *testing.T) {
}

func TestSelectUPFAndAllocUEIP(t *testing.T) {
	var expectedIPPool []net.IP

	for i := 16; i <= 31; i++ {
		expectedIPPool = append(expectedIPPool, net.ParseIP(fmt.Sprintf("10.60.0.%d", i)).To4())
	}

	userplaneInformation := NewUserPlaneInformation(configuration)

	for i := 0; i <= 100; i++ {
		upf, allocatedIP, _ := userplaneInformation.SelectUPFAndAllocUEIP(&UPFSelectionParams{
			Dnn: "internet",
			SNssai: &SNssai{
				Sst: 1,
				Sd:  "112232",
			},
		})

		require.Contains(t, expectedIPPool, allocatedIP)
		userplaneInformation.ReleaseUEIP(upf, allocatedIP, false)
	}
}
