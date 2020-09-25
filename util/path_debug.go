//+build debug

package util

import (
	"bitbucket.org/free5gc-team/path_util"
)

var SmfLogPath = path_util.Free5gcPath("free5gc/smfsslkey.log")
var SmfPemPath = path_util.Free5gcPath("free5gc/support/TLS/_debug.pem")
var SmfKeyPath = path_util.Free5gcPath("free5gc/support/TLS/_debug.key")
var DefaultSmfConfigPath = path_util.Free5gcPath("free5gc/config/smfcfg.conf")
