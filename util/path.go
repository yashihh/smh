//+build !debug

package util

import (
	"bitbucket.org/free5gc-team/path_util"
)

var (
	SmfLogPath           = path_util.Free5gcPath("free5gc/smfsslkey.log")
	SmfPemPath           = path_util.Free5gcPath("free5gc/support/TLS/smf.pem")
	SmfKeyPath           = path_util.Free5gcPath("free5gc/support/TLS/smf.key")
	DefaultSmfConfigPath = path_util.Free5gcPath("free5gc/config/smfcfg.yaml")
)
