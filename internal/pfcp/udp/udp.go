package udp

import (
	"net"
	"runtime/debug"
	"time"

	"bitbucket.org/free5gc-team/pfcp"
	"bitbucket.org/free5gc-team/pfcp/pfcpUdp"
	"bitbucket.org/free5gc-team/smf/internal/context"
	"bitbucket.org/free5gc-team/smf/internal/logger"
)

const MaxPfcpUdpDataSize = 1024

var Server *pfcpUdp.PfcpServer

var ServerStartTime time.Time

func Run(Dispatch func(*pfcpUdp.Message)) {
	defer func() {
		if p := recover(); p != nil {
			// Print stack for panic to log. Fatalf() will let program exit.
			logger.PfcpLog.Fatalf("panic: %v\n%s", p, string(debug.Stack()))
		}
	}()

	serverIP := context.SMF_Self().ExposeIP().To4()
	Server = pfcpUdp.NewPfcpServer(serverIP.String())

	err := Server.Listen()
	if err != nil {
		logger.PfcpLog.Errorf("Failed to listen: %v", err)
	}

	logger.PfcpLog.Infof("Listen on %s", Server.Conn.LocalAddr().String())

	go func(p *pfcpUdp.PfcpServer) {
		defer func() {
			if p := recover(); p != nil {
				// Print stack for panic to log. Fatalf() will let program exit.
				logger.PfcpLog.Fatalf("panic: %v\n%s", p, string(debug.Stack()))
			}
		}()

		for {
			var pfcpMessage pfcp.Message
			remoteAddr, err := p.ReadFrom(&pfcpMessage)
			if err != nil {
				if err.Error() == "Receive resend PFCP request" {
					logger.PfcpLog.Infoln(err)
				} else {
					logger.PfcpLog.Warnf("Read PFCP error: %v", err)
				}

				continue
			}

			msg := pfcpUdp.NewMessage(remoteAddr, &pfcpMessage)
			go Dispatch(&msg)
		}
	}(Server)

	ServerStartTime = time.Now()
}

func SendPfcp(msg pfcp.Message, addr *net.UDPAddr) {
	err := Server.WriteTo(msg, addr)
	if err != nil {
		logger.PfcpLog.Errorf("Failed to send PFCP message: %v", err)
	}
}
