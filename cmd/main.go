package main

import (
	"math/rand"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/urfave/cli"

	"bitbucket.org/free5gc-team/smf/internal/logger"
	"bitbucket.org/free5gc-team/smf/pkg/factory"
	"bitbucket.org/free5gc-team/smf/pkg/service"
	logger_util "bitbucket.org/free5gc-team/util/logger"
	"bitbucket.org/free5gc-team/util/version"
)

var SMF *service.SmfApp

func main() {
	defer func() {
		if p := recover(); p != nil {
			// Print stack for panic to log. Fatalf() will let program exit.
			logger.MainLog.Fatalf("panic: %v\n%s", p, string(debug.Stack()))
		}
	}()

	app := cli.NewApp()
	app.Name = "smf"
	app.Usage = "5G Session Management Function (SMF)"
	app.Action = action
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config, c",
			Usage: "Load configuration from `FILE`",
		},
		cli.StringFlag{
			Name:  "uerouting, u",
			Usage: "Load uerouting configuration from `FILE`",
		},
		cli.StringSliceFlag{
			Name:  "log, l",
			Usage: "Output NF log to `FILE`",
		},
	}
	rand.Seed(time.Now().UnixNano())

	if err := app.Run(os.Args); err != nil {
		logger.MainLog.Errorf("SMF Run error: %v\n", err)
	}
}

func action(cliCtx *cli.Context) error {
	tlsKeyLogPath, err := initLogFile(cliCtx.StringSlice("log"))
	if err != nil {
		return err
	}

	logger.MainLog.Infoln("SMF version: ", version.GetVersion())

	cfg, err := factory.ReadConfig(cliCtx.String("config"))
	if err != nil {
		return err
	}
	factory.SmfConfig = cfg

	smf, err := service.NewApp(cfg)
	if err != nil {
		return err
	}
	SMF = smf

	smf.Start(tlsKeyLogPath)

	return nil
}

func initLogFile(logNfPath []string) (string, error) {
	logTlsKeyPath := ""

	for _, path := range logNfPath {
		if err := logger_util.LogFileHook(logger.Log, path); err != nil {
			return "", err
		}

		if logTlsKeyPath != "" {
			continue
		}

		nfDir, _ := filepath.Split(path)
		tmpDir := filepath.Join(nfDir, "key")
		if err := os.MkdirAll(tmpDir, 0o775); err != nil {
			logger.InitLog.Errorf("Make directory %s failed: %+v", tmpDir, err)
			return "", err
		}
		_, name := filepath.Split(factory.SmfDefaultTLSKeyLogPath)
		logTlsKeyPath = filepath.Join(tmpDir, name)
	}

	return logTlsKeyPath, nil
}
