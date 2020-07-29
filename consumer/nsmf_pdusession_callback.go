package consumer

import (
	"context"
	"free5gc/lib/openapi"
	"free5gc/lib/openapi/Nsmf_PDUSession"
	"free5gc/lib/openapi/models"
	"free5gc/src/smf/logger"
	"net/http"
)

func SendSMContextStatusNotification(uri string) (problemDetails *models.ProblemDetails, err error) {
	if uri != "" {
		request := models.SmContextStatusNotification{}
		request.StatusInfo = &models.StatusInfo{
			ResourceStatus: models.ResourceStatus_RELEASED,
		}
		configuration := Nsmf_PDUSession.NewConfiguration()
		client := Nsmf_PDUSession.NewAPIClient(configuration)

		logger.CtxLog.Infoln("[SMF] Send SMContext Status Notification")
		httpResp, localErr := client.IndividualSMContextNotificationApi.SMContextNotification(context.Background(), uri, request)

		if localErr == nil {
			if httpResp.StatusCode != http.StatusNoContent {
				err = openapi.ReportError("Send SMContextStatus Notification Failed")
				return
			}

			logger.PduSessLog.Tracef("Send SMContextStatus Notification Success")
		} else if httpResp != nil {
			logger.PduSessLog.Warnf("Send SMContextStatus Notification Error[%s]", httpResp.Status)
			if httpResp.Status != localErr.Error() {
				err = localErr
				return
			}
			problem := localErr.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails)
			problemDetails = &problem
		} else {
			logger.PduSessLog.Warnln("Http Response is nil in comsumer API SMContextNotification")
			err = openapi.ReportError("Send SMContextStatus Notification Failed[%s]", localErr.Error())
		}

	}
	return
}
