package context_test

import (
	"fmt"
	"net"
	"testing"

	"bitbucket.org/free5gc-team/smf/context"
	"github.com/stretchr/testify/require"
)

func TestIPAddrWithOffset(t *testing.T) {
	var testcases = []struct {
		name         string
		inAddr       string
		inOffset     int
		expectedAddr string
	}{
		{
			name:         "add1",
			inAddr:       "60.60.0.0",
			inOffset:     1,
			expectedAddr: "60.60.0.1",
		},
		{
			name:         "add255",
			inAddr:       "60.60.0.0",
			inOffset:     255,
			expectedAddr: "60.60.0.255",
		},
		{
			name:         "add256",
			inAddr:       "60.60.0.0",
			inOffset:     256,
			expectedAddr: "60.60.1.0",
		},
		{
			name:         "add65536",
			inAddr:       "60.60.0.0",
			inOffset:     65536,
			expectedAddr: "60.61.0.0",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ip := net.ParseIP(tc.inAddr)
			retIP := context.IPAddrWithOffset(ip, tc.inOffset)
			require.Equal(t, tc.expectedAddr, retIP.String())
		})
	}
}

func TestIPAddrOffset(t *testing.T) {
	var testcases = []struct {
		name           string
		inAddr         string
		baseAddr       string
		expectedOffset int
	}{
		{
			name:           "diff1",
			baseAddr:       "60.60.0.0",
			expectedOffset: 1,
			inAddr:         "60.60.0.1",
		},
		{
			name:           "diff255",
			baseAddr:       "60.60.0.0",
			expectedOffset: 255,
			inAddr:         "60.60.0.255",
		},
		{
			name:           "diff256",
			baseAddr:       "60.60.0.0",
			expectedOffset: 256,
			inAddr:         "60.60.1.0",
		},
		{
			name:           "diff65536",
			baseAddr:       "60.60.0.0",
			expectedOffset: 65536,
			inAddr:         "60.61.0.0",
		},
		{
			name:           "diff68630",
			baseAddr:       "60.60.0.0",
			expectedOffset: 68630,
			inAddr:         "60.61.12.22",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ip := net.ParseIP(tc.inAddr)
			baseIP := net.ParseIP(tc.baseAddr)
			offset := context.IPAddrOffset(ip, baseIP)
			require.Equal(t, tc.expectedOffset, offset)
		})
	}
}

func TestIPAllocator(t *testing.T) {
	t.Run("Allocate", func(t *testing.T) {
		var err error
		var alloc *context.IPAllocator
		var ip net.IP
		alloc, err = context.NewIPAllocator("60.60.0.0/24")
		require.Nil(t, err)

		for i := 1; i <= 254; i++ {
			ip, err = alloc.Allocate()
			require.Nil(t, err)
			require.Equal(t, net.ParseIP(fmt.Sprintf("60.60.0.%d", i)).To4(), ip)
		}

		ip, err = alloc.Allocate()
		require.NotNil(t, err)
		require.Nil(t, ip)

	})

	t.Run("Release", func(t *testing.T) {
		var err error
		var alloc *context.IPAllocator
		var ip net.IP
		alloc, err = context.NewIPAllocator("60.60.0.0/24")
		require.Nil(t, err)

		for i := 1; i <= 254; i++ {
			ip, err = alloc.Allocate()
			require.Nil(t, err)
			require.Equal(t, net.ParseIP(fmt.Sprintf("60.60.0.%d", i)).To4(), ip)
		}

		ip, err = alloc.Allocate()
		require.NotNil(t, err)
		require.Nil(t, ip)

		alloc.Release(net.ParseIP("60.60.0.22").To4())

		ip, err = alloc.Allocate()
		require.Nil(t, err)
		require.Equal(t, net.ParseIP("60.60.0.22").To4(), ip)
	})
}
