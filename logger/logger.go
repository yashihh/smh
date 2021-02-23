package logger

import (
	"fmt"
	"os"
	"time"

	formatter "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"

	"bitbucket.org/free5gc-team/logger_util"
)

var (
	log         *logrus.Logger
	AppLog      *logrus.Entry
	InitLog     *logrus.Entry
	CfgLog      *logrus.Entry
	GsmLog      *logrus.Entry
	PfcpLog     *logrus.Entry
	PduSessLog  *logrus.Entry
	CtxLog      *logrus.Entry
	ConsumerLog *logrus.Entry
	GinLog      *logrus.Entry
)

func Initialize(logNf string, log5gc string) error {
	log = logrus.New()
	log.SetReportCaller(false)

	log.Formatter = &formatter.Formatter{
		TimestampFormat: time.RFC3339,
		TrimMessages:    true,
		NoFieldsSpace:   true,
		HideKeys:        true,
		FieldsOrder:     []string{"component", "category"},
	}

	if file, fullPath := logger_util.SetupLogger(log5gc, "", ""); file != "" {
		free5gcLogHook, err := logger_util.NewFileHook(fullPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o666)
		if err != nil {
			return err
		}
		log.Hooks.Add(free5gcLogHook)
	}

	if file, fullPath := logger_util.SetupLogger(logNf, "nf", "smf.log"); file != "" {
		if err := logger_util.ReFileName(fullPath); err != nil {
			fmt.Fprintf(os.Stderr, "Rename error: %v\n", err)
			return err
		}

		selfLogHook, err := logger_util.NewFileHook(fullPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o666)
		if err != nil {
			return err
		}
		log.Hooks.Add(selfLogHook)
	}

	AppLog = log.WithFields(logrus.Fields{"component": "SMF", "category": "App"})
	InitLog = log.WithFields(logrus.Fields{"component": "SMF", "category": "Init"})
	CfgLog = log.WithFields(logrus.Fields{"component": "SMF", "category": "CFG"})
	PfcpLog = log.WithFields(logrus.Fields{"component": "SMF", "category": "PFCP"})
	PduSessLog = log.WithFields(logrus.Fields{"component": "SMF", "category": "PduSess"})
	GsmLog = log.WithFields(logrus.Fields{"component": "SMF", "category": "GSM"})
	CtxLog = log.WithFields(logrus.Fields{"component": "SMF", "category": "CTX"})
	ConsumerLog = log.WithFields(logrus.Fields{"component": "SMF", "category": "Consumer"})
	GinLog = log.WithFields(logrus.Fields{"component": "SMF", "category": "GIN"})

	return nil
}

func SetLogLevel(level logrus.Level) {
	log.SetLevel(level)
}

func SetReportCaller(set bool) {
	log.SetReportCaller(set)
}
