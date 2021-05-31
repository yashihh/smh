package context

import (
	"fmt"
	"math/rand"
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"bitbucket.org/free5gc-team/smf/factory"
)

func TestUeIPPool(t *testing.T) {
	ueIPPool := NewUEIPPool(&factory.UEIPPool{
		Cidr: "10.10.0.0/24",
	})

	require.NotNil(t, ueIPPool)

	var allocIP net.IP

	// make allowed ip pools
	var ipPoolList []net.IP
	for i := 1; i < 255; i += 1 {
		ipStr := fmt.Sprintf("10.10.0.%d", i)
		ipPoolList = append(ipPoolList, net.ParseIP(ipStr).To4())
	}

	// allocate
	for i := 1; i < 255; i += 1 {
		allocIP = ueIPPool.allocate(nil)
		require.Contains(t, ipPoolList, allocIP)
	}

	// ip pool is empty
	allocIP = ueIPPool.allocate(nil)
	require.Nil(t, allocIP)

	// release IP
	for _, i := range rand.Perm(254) {
		ueIPPool.release(ipPoolList[i])
	}

	// allocate specify ip
	for _, ip := range ipPoolList {
		allocIP = ueIPPool.allocate(ip)
		require.Equal(t, ip, allocIP)
	}
}
