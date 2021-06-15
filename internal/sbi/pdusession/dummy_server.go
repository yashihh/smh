package pdusession

import (
	"log"
	"net/http"

	"bitbucket.org/free5gc-team/http2_util"
	"bitbucket.org/free5gc-team/logger_util"
	"bitbucket.org/free5gc-team/smf/internal/logger"
	"bitbucket.org/free5gc-team/smf/internal/pfcp"
	"bitbucket.org/free5gc-team/smf/internal/pfcp/udp"
)

func DummyServer() {
	router := logger_util.NewGinWithLogrus(logger.GinLog)

	AddService(router)

	go udp.Run(pfcp.Dispatch)

	smfKeyLogPath := "./log/smfsslkey.log"
	smfPemPath := "./config/TLS/smf.pem"
	smfkeyPath := "./config/TLS/smf.key"

	var server *http.Server
	if srv, err := http2_util.NewServer(":29502", smfKeyLogPath, router); err != nil {
	} else {
		server = srv
	}

	if err := server.ListenAndServeTLS(smfPemPath, smfkeyPath); err != nil {
		log.Fatal(err)
	}
}
