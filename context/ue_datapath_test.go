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
	fmt.Println("Start.")
	var testcases = []struct {
		name                  string
		inSUPI                string
		inPaths               []factory.Path
		expectedDataPathNodes []*context.DataPathNode
	}{
		{
			name:   "singleUPF",
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
			expectedDataPathNodes: []*context.DataPathNode{
				buildUeDataPathNode("UPF1"),
			},
		},
		{
			name:   "multiUPF",
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
			expectedDataPathNodes: []*context.DataPathNode{
				buildUeDataPathNode("UPF1"),
				buildUeDataPathNode("UPF2"),
			},
		},
	}

	// set the UpLinkTunnel and DownLinkTunnel of expectedDataPathNode
	for _, tc := range testcases {
		for index, node := range tc.expectedDataPathNodes {
			fmt.Printf("index: %d. length: %d.\n", index, len(tc.expectedDataPathNodes))
			if index != 0 {
				node.AddPrev(tc.expectedDataPathNodes[index-1])
			}
			if index != len(tc.expectedDataPathNodes)-1 {
				node.AddNext(tc.expectedDataPathNodes[index+1])
			}
		}
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			retUePreConfigPaths, err := context.NewUEPreConfigPaths(tc.inSUPI, tc.inPaths)
			require.Nil(t, err)
			require.NotNil(t, retUePreConfigPaths.PathIDGenerator)
			for index, path := range tc.inPaths {
				retDataPath := retUePreConfigPaths.DataPathPool[int64(index+1)]
				require.Equal(t, path.DestinationIP, retDataPath.Destination.DestinationIP)
				require.Equal(t, path.DestinationPort, retDataPath.Destination.DestinationPort)
				retNode := retDataPath.FirstDPNode
				for _, upfName := range path.UPF {
					expectedUpf := context.SMF_Self().UserPlaneInformation.UPNodes[upfName].UPF
					require.Equal(t, retNode.UPF, expectedUpf)
					retNode = retNode.DownLinkTunnel.SrcEndPoint
				}
				require.Nil(t, retNode)
			}
		})
	}
}

func buildUeDataPathNode(nodeName string) *context.DataPathNode {
	newUeNode, err := context.NewUEDataPathNode(nodeName)

	if err != nil {
		return nil
	}

	return newUeNode
}
