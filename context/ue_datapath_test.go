package context_test

import (
	"fmt"
	"testing"

	"bitbucket.org/free5gc-team/smf/context"
	"bitbucket.org/free5gc-team/smf/factory"
	"github.com/stretchr/testify/require"
)

var config = configuration

// context.smfContext.UserPlaneInformation = context.NewUserPlaneInformation(config)

func TestNewUEPreConfigPaths(t *testing.T) {
	smfContext := context.SMF_Self()
	smfContext.UserPlaneInformation = context.NewUserPlaneInformation(config)
	fmt.Println("Start")
	var testcases = []struct {
		name                  string
		inSUPI                string
		inPaths               []factory.Path
		expectedDataPathNodes [][]*context.UPF
	}{
		{
			name:   "singlePath-singleUPF",
			inSUPI: "imsi-2089300007487",
			inPaths: []factory.Path{
				{
					DestinationIP:   "60.60.0.101",
					DestinationPort: "12345",
					UPF: []string{
						"UPF1",
					},
				},
			},
			expectedDataPathNodes: [][]*context.UPF{
				{
					getUpf("UPF1"),
				},
			},
		},
		{
			name:   "singlePath-multiUPF",
			inSUPI: "imsi-2089300007487",
			inPaths: []factory.Path{
				{
					DestinationIP:   "60.60.0.101",
					DestinationPort: "12345",
					UPF: []string{
						"UPF1",
						"UPF2",
					},
				},
			},
			expectedDataPathNodes: [][]*context.UPF{
				{
					getUpf("UPF1"),
					getUpf("UPF2"),
				},
			},
		},
		{
			name:   "multiPath-singleUPF",
			inSUPI: "imsi-2089300007487",
			inPaths: []factory.Path{
				{
					DestinationIP:   "60.60.0.101",
					DestinationPort: "12345",
					UPF: []string{
						"UPF1",
					},
				},
				{
					DestinationIP:   "60.60.0.103",
					DestinationPort: "12345",
					UPF: []string{
						"UPF2",
					},
				},
			},
			expectedDataPathNodes: [][]*context.UPF{
				{
					getUpf("UPF1"),
				},
				{
					getUpf("UPF2"),
				},
			},
		},
		{
			name:   "multiPath-multiUPF",
			inSUPI: "imsi-2089300007487",
			inPaths: []factory.Path{
				{
					DestinationIP:   "60.60.0.101",
					DestinationPort: "12345",
					UPF: []string{
						"UPF1",
						"UPF2",
					},
				},
				{
					DestinationIP:   "60.60.0.103",
					DestinationPort: "12345",
					UPF: []string{
						"UPF1",
						"UPF3",
					},
				},
			},
			expectedDataPathNodes: [][]*context.UPF{
				{
					getUpf("UPF1"),
					getUpf("UPF2"),
				},
				{
					getUpf("UPF1"),
					getUpf("UPF3"),
				},
			},
		},
		{
			name:   "multiPath-single&multiUPF",
			inSUPI: "imsi-2089300007487",
			inPaths: []factory.Path{
				{
					DestinationIP:   "60.60.0.101",
					DestinationPort: "12345",
					UPF: []string{
						"UPF1",
					},
				},
				{
					DestinationIP:   "60.60.0.103",
					DestinationPort: "12345",
					UPF: []string{
						"UPF1",
						"UPF3",
					},
				},
			},
			expectedDataPathNodes: [][]*context.UPF{
				{
					getUpf("UPF1"),
				},
				{
					getUpf("UPF1"),
					getUpf("UPF3"),
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			retUePreConfigPaths, err := context.NewUEPreConfigPaths(tc.inSUPI, tc.inPaths)
			require.Nil(t, err)
			require.NotNil(t, retUePreConfigPaths.PathIDGenerator)
			for pathIndex, path := range tc.inPaths {
				retDataPath := retUePreConfigPaths.DataPathPool[int64(pathIndex+1)]
				require.Equal(t, path.DestinationIP, retDataPath.Destination.DestinationIP)
				require.Equal(t, path.DestinationPort, retDataPath.Destination.DestinationPort)
				retNode := retDataPath.FirstDPNode
				for _, expectedUpf := range tc.expectedDataPathNodes[pathIndex] {
					require.NotNil(t, retNode.UPF)
					require.Equal(t, retNode.UPF, expectedUpf)
					retNode = retNode.DownLinkTunnel.SrcEndPoint
				}
				require.Nil(t, retNode)
			}
		})
	}
}

func getUpf(name string) *context.UPF {
	newUeNode, err := context.NewUEDataPathNode(name)

	if err != nil {
		return nil
	}

	Upf := newUeNode.UPF

	return Upf
}
