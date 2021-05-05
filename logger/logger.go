package logger

import (
	"os"
	"time"

	formatter "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"

	aperLogger "bitbucket.org/free5gc-team/aper/logger"
	"bitbucket.org/free5gc-team/logger_util"
	nasLogger "bitbucket.org/free5gc-team/nas/logger"
	ngapLogger "bitbucket.org/free5gc-team/ngap/logger"
	NamfCommLogger "bitbucket.org/free5gc-team/openapi/Namf_Communication/logger"
	NamfEventLogger "bitbucket.org/free5gc-team/openapi/Namf_EventExposure/logger"
	NnssfNSSAIAvailabilityLogger "bitbucket.org/free5gc-team/openapi/Nnssf_NSSAIAvailability/logger"
	NnssfNSSelectionLogger "bitbucket.org/free5gc-team/openapi/Nnssf_NSSelection/logger"
	NsmfEventLogger "bitbucket.org/free5gc-team/openapi/Nsmf_EventExposure/logger"
	NsmfPDUSessionLogger "bitbucket.org/free5gc-team/openapi/Nsmf_PDUSession/logger"
	NudmEventLogger "bitbucket.org/free5gc-team/openapi/Nudm_EventExposure/logger"
	NudmParameterProvisionLogger "bitbucket.org/free5gc-team/openapi/Nudm_ParameterProvision/logger"
	NudmSubDataManagementLogger "bitbucket.org/free5gc-team/openapi/Nudm_SubscriberDataManagement/logger"
	NudmUEAuthLogger "bitbucket.org/free5gc-team/openapi/Nudm_UEAuthentication/logger"
	NudmUEContextManagLogger "bitbucket.org/free5gc-team/openapi/Nudm_UEContextManagement/logger"
	NudrDataRepositoryLogger "bitbucket.org/free5gc-team/openapi/Nudr_DataRepository/logger"
	openApiLogger "bitbucket.org/free5gc-team/openapi/logger"
	pfcpLogger "bitbucket.org/free5gc-team/pfcp/logger"
)

const (
	FieldSupi         = "supi"
	FieldPDUSessionID = "pdu_session_id"
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

func init() {
	log = logrus.New()
	log.SetReportCaller(false)

	log.Formatter = &formatter.Formatter{
		TimestampFormat: time.RFC3339,
		TrimMessages:    true,
		NoFieldsSpace:   true,
		HideKeys:        true,
		FieldsOrder:     []string{"component", "category"},
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
}

func LogFileHook(logNfPath string, log5gcPath string) error {
	if fileErr, fullPath := logger_util.CreateFree5gcLogFile(log5gcPath); fileErr == nil {
		if fullPath != "" {
			free5gcLogHook, err := logger_util.NewFileHook(fullPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o666)
			if err != nil {
				return err
			}
			log.Hooks.Add(free5gcLogHook)
			aperLogger.GetLogger().Hooks.Add(free5gcLogHook)
			ngapLogger.GetLogger().Hooks.Add(free5gcLogHook)
			nasLogger.GetLogger().Hooks.Add(free5gcLogHook)
			openApiLogger.GetLogger().Hooks.Add(free5gcLogHook)
			NamfCommLogger.GetLogger().Hooks.Add(free5gcLogHook)
			NamfEventLogger.GetLogger().Hooks.Add(free5gcLogHook)
			NnssfNSSAIAvailabilityLogger.GetLogger().Hooks.Add(free5gcLogHook)
			NnssfNSSelectionLogger.GetLogger().Hooks.Add(free5gcLogHook)
			NsmfEventLogger.GetLogger().Hooks.Add(free5gcLogHook)
			NsmfPDUSessionLogger.GetLogger().Hooks.Add(free5gcLogHook)
			NudmEventLogger.GetLogger().Hooks.Add(free5gcLogHook)
			NudmParameterProvisionLogger.GetLogger().Hooks.Add(free5gcLogHook)
			NudmSubDataManagementLogger.GetLogger().Hooks.Add(free5gcLogHook)
			NudmUEAuthLogger.GetLogger().Hooks.Add(free5gcLogHook)
			NudmUEContextManagLogger.GetLogger().Hooks.Add(free5gcLogHook)
			NudrDataRepositoryLogger.GetLogger().Hooks.Add(free5gcLogHook)
			pfcpLogger.GetLogger().Hooks.Add(free5gcLogHook)
		}
	} else {
		return fileErr
	}

	if fileErr, fullPath := logger_util.CreateNfLogFile(logNfPath, "smf.log"); fileErr == nil {
		selfLogHook, err := logger_util.NewFileHook(fullPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o666)
		if err != nil {
			return err
		}
		log.Hooks.Add(selfLogHook)
		aperLogger.GetLogger().Hooks.Add(selfLogHook)
		ngapLogger.GetLogger().Hooks.Add(selfLogHook)
		nasLogger.GetLogger().Hooks.Add(selfLogHook)
		openApiLogger.GetLogger().Hooks.Add(selfLogHook)
		NamfCommLogger.GetLogger().Hooks.Add(selfLogHook)
		NamfEventLogger.GetLogger().Hooks.Add(selfLogHook)
		NnssfNSSAIAvailabilityLogger.GetLogger().Hooks.Add(selfLogHook)
		NnssfNSSelectionLogger.GetLogger().Hooks.Add(selfLogHook)
		NsmfEventLogger.GetLogger().Hooks.Add(selfLogHook)
		NsmfPDUSessionLogger.GetLogger().Hooks.Add(selfLogHook)
		NudmEventLogger.GetLogger().Hooks.Add(selfLogHook)
		NudmParameterProvisionLogger.GetLogger().Hooks.Add(selfLogHook)
		NudmSubDataManagementLogger.GetLogger().Hooks.Add(selfLogHook)
		NudmUEAuthLogger.GetLogger().Hooks.Add(selfLogHook)
		NudmUEContextManagLogger.GetLogger().Hooks.Add(selfLogHook)
		NudrDataRepositoryLogger.GetLogger().Hooks.Add(selfLogHook)
		pfcpLogger.GetLogger().Hooks.Add(selfLogHook)
	} else {
		return fileErr
	}

	return nil
}

func SetLogLevel(level logrus.Level) {
	log.SetLevel(level)
}

func SetReportCaller(set bool) {
	log.SetReportCaller(set)
}
