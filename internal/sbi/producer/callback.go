package producer

import (
	"context"
	"net/http"

	"bitbucket.org/free5gc-team/openapi/Nsmf_EventExposure"
	"bitbucket.org/free5gc-team/openapi/models"
	smf_context "bitbucket.org/free5gc-team/smf/internal/context"
	"bitbucket.org/free5gc-team/smf/internal/logger"
	"bitbucket.org/free5gc-team/util/httpwrapper"
)

func HandleSMPolicyUpdateNotify(smContextRef string, request models.SmPolicyNotification) *httpwrapper.Response {
	logger.PduSessLog.Infoln("In HandleSMPolicyUpdateNotify")
	decision := request.SmPolicyDecision
	smContext := smf_context.GetSMContextByRef(smContextRef)

	if smContext == nil {
		logger.PduSessLog.Errorf("SMContext[%s] not found", smContextRef)
		httpResponse := httpwrapper.NewResponse(http.StatusBadRequest, nil, nil)
		return httpResponse
	}

	if decision.TsnPortManContDstt != nil {
		nasPdu, _ := smf_context.BuildGSMPDUSessionModificationCommand(smContext, decision.TsnPortManContDstt)
		if nasPdu != nil {
			logger.PduSessLog.Debugln("pdu session modification command for pmic: ", nasPdu)
			sendGSMPDUSessionModificationCommand(smContext, nasPdu)
		}
	}

	smContext.SMLock.Lock()
	defer smContext.SMLock.Unlock()

	smContext.CheckState(smf_context.Active)
	// Wait till the state becomes Active again
	// TODO: implement waiting in concurrent architecture

	smContext.SetState(smf_context.ModificationPending)

	// Update SessionRule from decision
	if err := smContext.ApplySessionRules(decision); err != nil {
		// TODO: Fill the error body
		smContext.Log.Errorf("SMPolicyUpdateNotify err: %v", err)
		return httpwrapper.NewResponse(http.StatusBadRequest, nil, nil)
	}

	//TODO: Response data type -
	//[200 OK] UeCampingRep
	//[200 OK] array(PartialSuccessReport)
	//[400 Bad Request] ErrorReport
	if err := smContext.ApplyPccRules(decision); err != nil {
		smContext.Log.Errorf("apply sm policy decision error: %+v", err)
		// TODO: Fill the error body
		return httpwrapper.NewResponse(http.StatusBadRequest, nil, nil)
	}

	smContext.SendUpPathChgNotification("EARLY", SendUpPathChgEventExposureNotification)
	// TODO
	nwtt_portnum := uint32(0)
	ActivateUPFSession(smContext, nil, nwtt_portnum)

	smContext.SendUpPathChgNotification("LATE", SendUpPathChgEventExposureNotification)

	smContext.PostRemoveDataPath()

	return httpwrapper.NewResponse(http.StatusNoContent, nil, nil)
}

func SendUpPathChgEventExposureNotification(
	uri string, notification *models.NsmfEventExposureNotification,
) {
	configuration := Nsmf_EventExposure.NewConfiguration()
	client := Nsmf_EventExposure.NewAPIClient(configuration)
	_, httpResponse, err := client.
		DefaultCallbackApi.
		SmfEventExposureNotification(context.Background(), uri, *notification)
	if err != nil {
		if httpResponse != nil {
			logger.PduSessLog.Warnf("SMF Event Exposure Notification Error[%s]", httpResponse.Status)
		} else {
			logger.PduSessLog.Warnf("SMF Event Exposure Notification Failed[%s]", err.Error())
		}
		return
	} else if httpResponse == nil {
		logger.PduSessLog.Warnln("SMF Event Exposure Notification Failed[HTTP Response is nil]")
		return
	}
	defer func() {
		if rspCloseErr := httpResponse.Body.Close(); rspCloseErr != nil {
			logger.PduSessLog.Errorf("SmfEventExposureNotification response body cannot close: %+v", rspCloseErr)
		}
	}()
	if httpResponse.StatusCode != http.StatusOK && httpResponse.StatusCode != http.StatusNoContent {
		logger.PduSessLog.Warnf("SMF Event Exposure Notification Failed")
	} else {
		logger.PduSessLog.Tracef("SMF Event Exposure Notification Success")
	}
}
