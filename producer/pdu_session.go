package producer

import (
	"context"
	"encoding/hex"
	"net"
	"net/http"

	"github.com/antihax/optional"

	"bitbucket.org/free5gc-team/http_wrapper"
	"bitbucket.org/free5gc-team/nas"
	"bitbucket.org/free5gc-team/nas/nasMessage"
	"bitbucket.org/free5gc-team/openapi"
	"bitbucket.org/free5gc-team/openapi/Namf_Communication"
	"bitbucket.org/free5gc-team/openapi/Nsmf_PDUSession"
	"bitbucket.org/free5gc-team/openapi/Nudm_SubscriberDataManagement"
	"bitbucket.org/free5gc-team/openapi/models"
	"bitbucket.org/free5gc-team/pfcp/pfcpType"
	"bitbucket.org/free5gc-team/smf/consumer"
	smf_context "bitbucket.org/free5gc-team/smf/context"
	"bitbucket.org/free5gc-team/smf/logger"
	pfcp_message "bitbucket.org/free5gc-team/smf/pfcp/message"
)

func HandlePDUSessionSMContextCreate(request models.PostSmContextsRequest) *http_wrapper.Response {
	// GSM State
	// PDU Session Establishment Accept/Reject
	var response models.PostSmContextsResponse
	response.JsonData = new(models.SmContextCreatedData)
	logger.PduSessLog.Infoln("In HandlePDUSessionSMContextCreate")

	// Check has PDU Session Establishment Request
	m := nas.NewMessage()
	if err := m.GsmMessageDecode(&request.BinaryDataN1SmMessage); err != nil ||
		m.GsmHeader.GetMessageType() != nas.MsgTypePDUSessionEstablishmentRequest {
		logger.PduSessLog.Warnln("GsmMessageDecode Error: ", err)
		httpResponse := &http_wrapper.Response{
			Header: nil,
			Status: http.StatusForbidden,
			Body: models.PostSmContextsErrorResponse{
				JsonData: &models.SmContextCreateError{
					Error: &Nsmf_PDUSession.N1SmError,
				},
			},
		}
		return httpResponse
	}

	createData := request.JsonData
	smContext := smf_context.NewSMContext(createData.Supi, createData.PduSessionId)
	smContext.SetState(smf_context.ActivePending)
	smContext.SetCreateData(createData)
	smContext.SmStatusNotifyUri = createData.SmContextStatusUri

	// TODO: This lock should be unlock after PFPF Establishment,
	//       but this is workaround we must handle the pfcp timeout case to prevent SM lock allways lock
	smContext.SMLock.Lock()

	// DNN Information from config
	smContext.DNNInfo = smf_context.RetrieveDnnInformation(smContext.Snssai, smContext.Dnn)
	if smContext.DNNInfo == nil {
		logger.PduSessLog.Errorf("S-NSSAI[sst: %d, sd: %s] DNN[%s] not matched DNN Config",
			smContext.Snssai.Sst, smContext.Snssai.Sd, smContext.Dnn)
	}
	smContext.Log.Debugf("S-NSSAI[sst: %d, sd: %s] DNN[%s]",
		smContext.Snssai.Sst, smContext.Snssai.Sd, smContext.Dnn)

	// Query UDM
	if problemDetails, err := consumer.SendNFDiscoveryUDM(); err != nil {
		smContext.Log.Warnf("Send NF Discovery Serving UDM Error[%v]", err)
	} else if problemDetails != nil {
		smContext.Log.Warnf("Send NF Discovery Serving UDM Problem[%+v]", problemDetails)
	} else {
		smContext.Log.Infoln("Send NF Discovery Serving UDM Successfully")
	}

	// IP Allocation
	upfSelectionParams := &smf_context.UPFSelectionParams{
		Dnn: smContext.Dnn,
		SNssai: &smf_context.SNssai{
			Sst: smContext.Snssai.Sst,
			Sd:  smContext.Snssai.Sd,
		},
	}
	var selectedUPF *smf_context.UPNode
	var ip net.IP
	selectedUPFName := ""
	if smf_context.SMF_Self().ULCLSupport && smf_context.CheckUEHasPreConfig(createData.Supi) {
		groupName := smf_context.GetULCLGroupNameFromSUPI(createData.Supi)
		defaultPathPool := smf_context.GetUEDefaultPathPool(groupName)
		if defaultPathPool != nil {
			selectedUPFName, ip = defaultPathPool.SelectUPFAndAllocUEIPForULCL(
				smf_context.GetUserPlaneInformation(), upfSelectionParams)
			selectedUPF = smf_context.GetUserPlaneInformation().UPFs[selectedUPFName]
		}
	} else {
		selectedUPF, ip = smf_context.GetUserPlaneInformation().SelectUPFAndAllocUEIP(upfSelectionParams)
		smContext.PDUAddress = ip
		smContext.Log.Infof("Allocated PDUAdress[%s]", smContext.PDUAddress.String())
	}

	if ip == nil && (selectedUPF == nil || selectedUPFName == "") {
		smContext.Log.Error("fail allocate PDU address")

		smContext.SetState(smf_context.InActive)
		smContext.Log.Warnf("Data Path not found\n")
		smContext.Log.Warnln("Selection Parameter: ", upfSelectionParams.String())

		var httpResponse *http_wrapper.Response
		if buf, err := smf_context.
			BuildGSMPDUSessionEstablishmentReject(
				smContext,
				nasMessage.Cause5GSMInsufficientResourcesForSpecificSliceAndDNN); err != nil {
			httpResponse = &http_wrapper.Response{
				Header: nil,
				Status: http.StatusForbidden,
				Body: models.PostSmContextsErrorResponse{
					JsonData: &models.SmContextCreateError{
						Error:   &Nsmf_PDUSession.InsufficientResourceSliceDnn,
						N1SmMsg: &models.RefToBinaryData{ContentId: "n1SmMsg"},
					},
				},
			}
		} else {
			httpResponse = &http_wrapper.Response{
				Header: nil,
				Status: http.StatusForbidden,
				Body: models.PostSmContextsErrorResponse{
					JsonData: &models.SmContextCreateError{
						Error:   &Nsmf_PDUSession.InsufficientResourceSliceDnn,
						N1SmMsg: &models.RefToBinaryData{ContentId: "n1SmMsg"},
					},
					BinaryDataN1SmMessage: buf,
				},
			}
		}

		return httpResponse
	}
	smContext.PDUAddress = ip
	smContext.SelectedUPF = selectedUPF

	smPlmnID := createData.Guami.PlmnId

	smDataParams := &Nudm_SubscriberDataManagement.GetSmDataParamOpts{
		Dnn:         optional.NewString(createData.Dnn),
		PlmnId:      optional.NewInterface(smPlmnID.Mcc + smPlmnID.Mnc),
		SingleNssai: optional.NewInterface(openapi.MarshToJsonString(smContext.Snssai)),
	}

	SubscriberDataManagementClient := smf_context.SMF_Self().SubscriberDataManagementClient

	if sessSubData, rsp, err := SubscriberDataManagementClient.
		SessionManagementSubscriptionDataRetrievalApi.
		GetSmData(context.Background(), smContext.Supi, smDataParams); err != nil {
		smContext.Log.Errorln("Get SessionManagementSubscriptionData error:", err)
	} else {
		defer func() {
			if rspCloseErr := rsp.Body.Close(); rspCloseErr != nil {
				smContext.Log.Errorf("GetSmData response body cannot close: %+v", rspCloseErr)
			}
		}()
		if len(sessSubData) > 0 {
			smContext.DnnConfiguration = sessSubData[0].DnnConfigurations[smContext.Dnn]
		} else {
			smContext.Log.Errorln("SessionManagementSubscriptionData from UDM is nil")
		}
	}

	establishmentRequest := m.PDUSessionEstablishmentRequest
	smContext.HandlePDUSessionEstablishmentRequest(establishmentRequest)

	if err := smContext.PCFSelection(); err != nil {
		smContext.Log.Errorln("pcf selection error:", err)
	}

	var smPolicyDecision *models.SmPolicyDecision
	if smPolicyDecisionRsp, err := consumer.SendSMPolicyAssociationCreate(smContext); err != nil {
		smContext.Log.Errorf("SM Policy Association failed")
	} else {
		smPolicyDecision = smPolicyDecisionRsp
	}

	// dataPath selection
	smContext.Tunnel = smf_context.NewUPTunnel()
	if err := ApplySmPolicyFromDecision(smContext, smPolicyDecision); err != nil {
		smContext.Log.Errorf("apply sm policy decision error: %+v", err)
	}
	var defaultPath *smf_context.DataPath

	if smf_context.SMF_Self().ULCLSupport && smf_context.CheckUEHasPreConfig(createData.Supi) {
		smContext.Log.Infof("Has pre-config route")
		uePreConfigPaths := smf_context.GetUEPreConfigPaths(createData.Supi, selectedUPFName)
		smContext.Tunnel.DataPathPool = uePreConfigPaths.DataPathPool
		smContext.Tunnel.PathIDGenerator = uePreConfigPaths.PathIDGenerator
		defaultPath = smContext.Tunnel.DataPathPool.GetDefaultPath()
		defaultPath.ActivateTunnelAndPDR(smContext, 255)
		smContext.BPManager = smf_context.NewBPManager(createData.Supi)
	} else {
		// UE has no pre-config path.
		// Use default route
		smContext.Log.Infof("Has no pre-config route")
		defaultUPPath := smf_context.GetUserPlaneInformation().GetDefaultUserPlanePathByDNNAndUPF(
			upfSelectionParams, smContext.SelectedUPF)
		defaultPath = smf_context.GenerateDataPath(defaultUPPath, smContext)
		if defaultPath != nil {
			defaultPath.IsDefaultPath = true
			smContext.Tunnel.AddDataPath(defaultPath)
			defaultPath.ActivateTunnelAndPDR(smContext, 255)
		}
	}

	if defaultPath == nil {
		smContext.SetState(smf_context.InActive)
		smContext.Log.Warnf("Data Path not found\n")
		smContext.Log.Warnln("Selection Parameter: ", upfSelectionParams.String())

		var httpResponse *http_wrapper.Response
		if buf, err := smf_context.
			BuildGSMPDUSessionEstablishmentReject(
				smContext,
				nasMessage.Cause5GSMInsufficientResourcesForSpecificSliceAndDNN); err != nil {
			httpResponse = &http_wrapper.Response{
				Header: nil,
				Status: http.StatusForbidden,
				Body: models.PostSmContextsErrorResponse{
					JsonData: &models.SmContextCreateError{
						Error:   &Nsmf_PDUSession.InsufficientResourceSliceDnn,
						N1SmMsg: &models.RefToBinaryData{ContentId: "n1SmMsg"},
					},
				},
			}
		} else {
			httpResponse = &http_wrapper.Response{
				Header: nil,
				Status: http.StatusForbidden,
				Body: models.PostSmContextsErrorResponse{
					JsonData: &models.SmContextCreateError{
						Error:   &Nsmf_PDUSession.InsufficientResourceSliceDnn,
						N1SmMsg: &models.RefToBinaryData{ContentId: "n1SmMsg"},
					},
					BinaryDataN1SmMessage: buf,
				},
			}
		}

		return httpResponse
	}

	if problemDetails, err := consumer.SendNFDiscoveryServingAMF(smContext); err != nil {
		smContext.Log.Warnf("Send NF Discovery Serving AMF Error[%v]", err)
	} else if problemDetails != nil {
		smContext.Log.Warnf("Send NF Discovery Serving AMF Problem[%+v]", problemDetails)
	} else {
		smContext.Log.Traceln("Send NF Discovery Serving AMF successfully")
	}

	for _, service := range *smContext.AMFProfile.NfServices {
		if service.ServiceName == models.ServiceName_NAMF_COMM {
			communicationConf := Namf_Communication.NewConfiguration()
			communicationConf.SetBasePath(service.ApiPrefix)
			smContext.CommunicationClient = Namf_Communication.NewAPIClient(communicationConf)
		}
	}

	SendPFCPRules(smContext)

	response.JsonData = smContext.BuildCreatedData()
	httpResponse := &http_wrapper.Response{
		Header: http.Header{
			"Location": {smContext.Ref},
		},
		Status: http.StatusCreated,
		Body:   response,
	}

	return httpResponse
	// TODO: UECM registration
}

func HandlePDUSessionSMContextUpdate(smContextRef string, body models.UpdateSmContextRequest) *http_wrapper.Response {
	// GSM State
	// PDU Session Modification Reject(Cause Value == 43 || Cause Value != 43)/Complete
	// PDU Session Release Command/Complete
	logger.PduSessLog.Infoln("In HandlePDUSessionSMContextUpdate")
	smContext := smf_context.GetSMContext(smContextRef)

	if smContext == nil {
		logger.PduSessLog.Warnf("SMContext[%s] is not found", smContextRef)

		httpResponse := &http_wrapper.Response{
			Header: nil,
			Status: http.StatusNotFound,
			Body: models.UpdateSmContextErrorResponse{
				JsonData: &models.SmContextUpdateError{
					UpCnxState: models.UpCnxState_DEACTIVATED,
					Error: &models.ProblemDetails{
						Type:   "Resource Not Found",
						Title:  "SMContext Ref is not found",
						Status: http.StatusNotFound,
					},
				},
			},
		}
		return httpResponse
	}

	smContext.SMLock.Lock()
	defer smContext.SMLock.Unlock()

	var sendPFCPDelete, sendPFCPModification bool
	var response models.UpdateSmContextResponse
	response.JsonData = new(models.SmContextUpdatedData)

	smContextUpdateData := body.JsonData

	if body.BinaryDataN1SmMessage != nil {
		m := nas.NewMessage()
		err := m.GsmMessageDecode(&body.BinaryDataN1SmMessage)
		smContext.Log.Tracef("N1 Message: %s", hex.EncodeToString(body.BinaryDataN1SmMessage))
		if err != nil {
			smContext.Log.Errorf("N1 Message parse failed: %v", err)
			httpResponse := &http_wrapper.Response{
				Status: http.StatusForbidden,
				Body: models.UpdateSmContextErrorResponse{
					JsonData: &models.SmContextUpdateError{
						Error: &Nsmf_PDUSession.N1SmError,
					},
				}, // Depends on the reason why N4 fail
			}
			return httpResponse
		}

		switch m.GsmHeader.GetMessageType() {
		case nas.MsgTypePDUSessionReleaseRequest:
			smContext.CheckState(smf_context.Active)
			// Wait till the state becomes Active again
			// TODO: implement sleep wait in concurrent architecture

			smContext.HandlePDUSessionReleaseRequest(m.PDUSessionReleaseRequest)
			if smContext.SelectedUPF != nil {
				smContext.Log.Infof("Release IP[%s]", smContext.PDUAddress.String())
				smf_context.GetUserPlaneInformation().ReleaseUEIP(smContext.SelectedUPF, smContext.PDUAddress)
			}

			if buf, err := smf_context.BuildGSMPDUSessionReleaseCommand(
				smContext, nasMessage.Cause5GSMRegularDeactivation); err != nil {
				smContext.Log.Errorf("Build GSM PDUSessionReleaseCommand failed: %+v", err)
			} else {
				response.BinaryDataN1SmMessage = buf
			}

			response.JsonData.N1SmMsg = &models.RefToBinaryData{ContentId: "PDUSessionReleaseCommand"}

			response.JsonData.N2SmInfo = &models.RefToBinaryData{ContentId: "PDUResourceReleaseCommand"}
			response.JsonData.N2SmInfoType = models.N2SmInfoType_PDU_RES_REL_CMD

			if buf, err := smf_context.BuildPDUSessionResourceReleaseCommandTransfer(smContext); err != nil {
				smContext.Log.Errorf("Build PDUSessionResourceReleaseCommandTransfer failed: %+v", err)
			} else {
				response.BinaryDataN2SmInformation = buf
			}

			smContext.SetState(smf_context.PFCPModification)

			releaseTunnel(smContext)

			sendPFCPDelete = true
		case nas.MsgTypePDUSessionReleaseComplete:
			smContext.CheckState(smf_context.InActivePending)
			// Wait till the state becomes Active again
			// TODO: implement sleep wait in concurrent architecture

			// Send Release Notify to AMF
			smContext.Log.Infoln("[SMF] Send Update SmContext Response")
			smContext.SetState(smf_context.InActive)
			response.JsonData.UpCnxState = models.UpCnxState_DEACTIVATED
			problemDetails, err := consumer.SendSMContextStatusNotification(smContext.SmStatusNotifyUri)
			if problemDetails != nil || err != nil {
				if problemDetails != nil {
					smContext.Log.Warnf("Send SMContext Status Notification Problem[%+v]", problemDetails)
				}

				if err != nil {
					smContext.Log.Warnf("Send SMContext Status Notification Error[%v]", err)
				}
			} else {
				smContext.Log.Traceln("Send SMContext Status Notification successfully")
			}
		}
	}

	tunnel := smContext.Tunnel
	pdrList := []*smf_context.PDR{}
	farList := []*smf_context.FAR{}
	barList := []*smf_context.BAR{}
	qerList := []*smf_context.QER{}

	switch smContextUpdateData.UpCnxState {
	case models.UpCnxState_ACTIVATING:
		smContext.CheckState(smf_context.Active)
		// Wait till the state becomes Active again
		// TODO: implement sleep wait in concurrent architecture

		smContext.SetState(smf_context.ModificationPending)
		response.JsonData.N2SmInfo = &models.RefToBinaryData{ContentId: "PDUSessionResourceSetupRequestTransfer"}
		response.JsonData.UpCnxState = models.UpCnxState_ACTIVATING
		response.JsonData.N2SmInfoType = models.N2SmInfoType_PDU_RES_SETUP_REQ

		n2Buf, err := smf_context.BuildPDUSessionResourceSetupRequestTransfer(smContext)
		if err != nil {
			smContext.Log.Errorf("Build PDUSession Resource Setup Request Transfer Error(%s)", err.Error())
		}
		smContext.UpCnxState = models.UpCnxState_ACTIVATING
		response.BinaryDataN2SmInformation = n2Buf
		response.JsonData.N2SmInfoType = models.N2SmInfoType_PDU_RES_SETUP_REQ
	case models.UpCnxState_DEACTIVATED:
		smContext.CheckState(smf_context.Active)
		// Wait till the state becomes Active again
		// TODO: implement sleep wait in concurrent architecture

		smContext.SetState(smf_context.ModificationPending)
		response.JsonData.UpCnxState = models.UpCnxState_DEACTIVATED
		smContext.UpCnxState = body.JsonData.UpCnxState
		smContext.UeLocation = body.JsonData.UeLocation

		// TODO: Deactivate N2 downlink tunnel
		// Set FAR and An, N3 Release Info
		farList = []*smf_context.FAR{}
		smContext.PendingUPF = make(smf_context.PendingUPF)
		for _, dataPath := range smContext.Tunnel.DataPathPool {
			ANUPF := dataPath.FirstDPNode
			DLPDR := ANUPF.DownLinkTunnel.PDR
			if DLPDR == nil {
				smContext.Log.Errorf("AN Release Error")
			} else {
				DLPDR.FAR.State = smf_context.RULE_UPDATE
				DLPDR.FAR.ApplyAction.Forw = false
				DLPDR.FAR.ApplyAction.Buff = true
				DLPDR.FAR.ApplyAction.Nocp = true
				smContext.PendingUPF[ANUPF.GetNodeIP()] = true
			}

			farList = append(farList, DLPDR.FAR)
		}

		sendPFCPModification = true
		smContext.SetState(smf_context.PFCPModification)
	}

	switch smContextUpdateData.N2SmInfoType {
	case models.N2SmInfoType_PDU_RES_SETUP_RSP:
		smContext.CheckState(smf_context.Active)
		// Wait till the state becomes Active again
		// TODO: implement sleep wait in concurrent architecture

		smContext.SetState(smf_context.ModificationPending)
		pdrList = []*smf_context.PDR{}
		farList = []*smf_context.FAR{}

		smContext.PendingUPF = make(smf_context.PendingUPF)
		for _, dataPath := range tunnel.DataPathPool {
			if dataPath.Activated {
				ANUPF := dataPath.FirstDPNode
				DLPDR := ANUPF.DownLinkTunnel.PDR

				DLPDR.FAR.ApplyAction = pfcpType.ApplyAction{Buff: false, Drop: false, Dupl: false, Forw: true, Nocp: false}
				DLPDR.FAR.ForwardingParameters = &smf_context.ForwardingParameters{
					DestinationInterface: pfcpType.DestinationInterface{
						InterfaceValue: pfcpType.DestinationInterfaceAccess,
					},
					NetworkInstance: []byte(smContext.Dnn),
				}

				DLPDR.State = smf_context.RULE_UPDATE
				DLPDR.FAR.State = smf_context.RULE_UPDATE

				pdrList = append(pdrList, DLPDR)
				farList = append(farList, DLPDR.FAR)

				if _, exist := smContext.PendingUPF[ANUPF.GetNodeIP()]; !exist {
					smContext.PendingUPF[ANUPF.GetNodeIP()] = true
				}
			}
		}

		if err := smf_context.
			HandlePDUSessionResourceSetupResponseTransfer(body.BinaryDataN2SmInformation, smContext); err != nil {
			smContext.Log.Errorf("Handle PDUSessionResourceSetupResponseTransfer failed: %+v", err)
		}
		sendPFCPModification = true
		smContext.SetState(smf_context.PFCPModification)
	case models.N2SmInfoType_PDU_RES_SETUP_FAIL:
		if err := smf_context.
			HandlePDUSessionResourceSetupUnsuccessfulTransfer(body.BinaryDataN2SmInformation, smContext); err != nil {
			smContext.Log.Errorf("Handle PDUSessionResourceSetupResponseTransfer failed: %+v", err)
		}
	case models.N2SmInfoType_PDU_RES_REL_RSP:
		smContext.Log.Infoln("[SMF] N2 PDUSession Release Complete")
		if smContext.PDUSessionRelease_DUE_TO_DUP_PDU_ID {
			smContext.CheckState(smf_context.InActivePending)
			// Wait till the state becomes Active again
			// TODO: implement sleep wait in concurrent architecture

			smContext.SetState(smf_context.InActive)
			smContext.Log.Infoln("[SMF] Send Update SmContext Response")
			response.JsonData.UpCnxState = models.UpCnxState_DEACTIVATED

			smContext.PDUSessionRelease_DUE_TO_DUP_PDU_ID = false
			smf_context.RemoveSMContext(smContext.Ref)
			problemDetails, err := consumer.SendSMContextStatusNotification(smContext.SmStatusNotifyUri)
			if problemDetails != nil || err != nil {
				if problemDetails != nil {
					smContext.Log.Warnf("Send SMContext Status Notification Problem[%+v]", problemDetails)
				}

				if err != nil {
					smContext.Log.Warnf("Send SMContext Status Notification Error[%v]", err)
				}
			} else {
				smContext.Log.Traceln("Send SMContext Status Notification successfully")
			}
		} else { // normal case
			smContext.CheckState(smf_context.InActivePending)
			// Wait till the state becomes Active again
			// TODO: implement sleep wait in concurrent architecture

			smContext.Log.Infoln("[SMF] Send Update SmContext Response")
			smContext.SetState(smf_context.InActivePending)
		}
	case models.N2SmInfoType_PATH_SWITCH_REQ:
		smContext.Log.Traceln("Handle Path Switch Request")
		smContext.CheckState(smf_context.Active)
		// Wait till the state becomes Active again
		// TODO: implement sleep wait in concurrent architecture

		smContext.SetState(smf_context.ModificationPending)

		if err := smf_context.HandlePathSwitchRequestTransfer(body.BinaryDataN2SmInformation, smContext); err != nil {
			smContext.Log.Errorf("Handle PathSwitchRequestTransfer: %+v", err)
		}

		if n2Buf, err := smf_context.BuildPathSwitchRequestAcknowledgeTransfer(smContext); err != nil {
			smContext.Log.Errorf("Build Path Switch Transfer Error(%+v)", err)
		} else {
			response.BinaryDataN2SmInformation = n2Buf
		}

		response.JsonData.N2SmInfoType = models.N2SmInfoType_PATH_SWITCH_REQ_ACK
		response.JsonData.N2SmInfo = &models.RefToBinaryData{
			ContentId: "PATH_SWITCH_REQ_ACK",
		}

		smContext.PendingUPF = make(smf_context.PendingUPF)
		for _, dataPath := range tunnel.DataPathPool {
			if dataPath.Activated {
				ANUPF := dataPath.FirstDPNode
				DLPDR := ANUPF.DownLinkTunnel.PDR

				pdrList = append(pdrList, DLPDR)
				farList = append(farList, DLPDR.FAR)

				if _, exist := smContext.PendingUPF[ANUPF.GetNodeIP()]; !exist {
					smContext.PendingUPF[ANUPF.GetNodeIP()] = true
				}
			}
		}

		sendPFCPModification = true
		smContext.SetState(smf_context.PFCPModification)
	case models.N2SmInfoType_PATH_SWITCH_SETUP_FAIL:
		smContext.CheckState(smf_context.Active)
		// Wait till the state becomes Active again
		// TODO: implement sleep wait in concurrent architecture

		smContext.SetState(smf_context.ModificationPending)
		if err :=
			smf_context.HandlePathSwitchRequestSetupFailedTransfer(body.BinaryDataN2SmInformation, smContext); err != nil {
			smContext.Log.Errorf("HandlePathSwitchRequestSetupFailedTransfer failed: %v", err)
		}
	case models.N2SmInfoType_HANDOVER_REQUIRED:
		smContext.CheckState(smf_context.Active)
		// Wait till the state becomes Active again
		// TODO: implement sleep wait in concurrent architecture
		smContext.SetState(smf_context.ModificationPending)
		response.JsonData.N2SmInfo = &models.RefToBinaryData{ContentId: "Handover"}
	}

	switch smContextUpdateData.HoState {
	case models.HoState_PREPARING:
		smContext.Log.Traceln("In HoState_PREPARING")
		smContext.CheckState(smf_context.Active)
		// Wait till the state becomes Active again
		// TODO: implement sleep wait in concurrent architecture

		smContext.SetState(smf_context.ModificationPending)
		smContext.HoState = models.HoState_PREPARING
		if err := smf_context.HandleHandoverRequiredTransfer(body.BinaryDataN2SmInformation, smContext); err != nil {
			smContext.Log.Errorf("Handle HandoverRequiredTransfer failed: %+v", err)
		}
		response.JsonData.N2SmInfoType = models.N2SmInfoType_PDU_RES_SETUP_REQ

		if n2Buf, err := smf_context.BuildPDUSessionResourceSetupRequestTransfer(smContext); err != nil {
			smContext.Log.Errorf("Build PDUSession Resource Setup Request Transfer Error(%s)", err.Error())
		} else {
			response.BinaryDataN2SmInformation = n2Buf
		}
		response.JsonData.N2SmInfoType = models.N2SmInfoType_PDU_RES_SETUP_REQ
		response.JsonData.N2SmInfo = &models.RefToBinaryData{
			ContentId: "PDU_RES_SETUP_REQ",
		}
		response.JsonData.HoState = models.HoState_PREPARING
	case models.HoState_PREPARED:
		smContext.Log.Traceln("In HoState_PREPARED")
		smContext.CheckState(smf_context.Active)
		// Wait till the state becomes Active again
		// TODO: implement sleep wait in concurrent architecture

		smContext.SetState(smf_context.ModificationPending)
		smContext.HoState = models.HoState_PREPARED
		response.JsonData.HoState = models.HoState_PREPARED
		if err :=
			smf_context.HandleHandoverRequestAcknowledgeTransfer(body.BinaryDataN2SmInformation, smContext); err != nil {
			smContext.Log.Errorf("Handle HandoverRequestAcknowledgeTransfer failed: %+v", err)
		}

		// request UPF establish indirect forwarding path for DL
		if smContext.DLForwardingType == smf_context.IndirectForwarding {
			smContext.PendingUPF = make(smf_context.PendingUPF)
			ANUPF := smContext.IndirectForwardingTunnel.FirstDPNode
			IndirectForwardingPDR := smContext.IndirectForwardingTunnel.FirstDPNode.UpLinkTunnel.PDR

			pdrList = append(pdrList, IndirectForwardingPDR)
			farList = append(farList, IndirectForwardingPDR.FAR)

			if _, exist := smContext.PendingUPF[ANUPF.GetNodeIP()]; !exist {
				smContext.PendingUPF[ANUPF.GetNodeIP()] = true
			}

			// release indirect forwading path
			if err := ANUPF.UPF.RemovePDR(IndirectForwardingPDR); err != nil {
				logger.PduSessLog.Errorln("release indirect path: ", err)
			}

			sendPFCPModification = true
			smContext.SetState(smf_context.PFCPModification)
		}

		if n2Buf, err := smf_context.BuildHandoverCommandTransfer(smContext); err != nil {
			smContext.Log.Errorf("Build HandoverCommandTransfer failed: %v", err)
		} else {
			response.BinaryDataN2SmInformation = n2Buf
		}

		response.JsonData.N2SmInfoType = models.N2SmInfoType_HANDOVER_CMD
		response.JsonData.N2SmInfo = &models.RefToBinaryData{
			ContentId: "HANDOVER_CMD",
		}
		response.JsonData.HoState = models.HoState_PREPARING
	case models.HoState_COMPLETED:
		smContext.Log.Traceln("In HoState_COMPLETED")
		smContext.CheckState(smf_context.Active)
		// Wait till the state becomes Active again
		// TODO: implement sleep wait in concurrent architecture

		smContext.PendingUPF = make(smf_context.PendingUPF)
		for _, dataPath := range tunnel.DataPathPool {
			if dataPath.Activated {
				ANUPF := dataPath.FirstDPNode
				DLPDR := ANUPF.DownLinkTunnel.PDR

				pdrList = append(pdrList, DLPDR)
				farList = append(farList, DLPDR.FAR)

				if _, exist := smContext.PendingUPF[ANUPF.GetNodeIP()]; !exist {
					smContext.PendingUPF[ANUPF.GetNodeIP()] = true
				}
			}
		}

		// remove indirect forwarding path
		if smContext.DLForwardingType == smf_context.IndirectForwarding {
			indirectForwardingPDR := smContext.IndirectForwardingTunnel.FirstDPNode.GetUpLinkPDR()
			indirectForwardingPDR.State = smf_context.RULE_REMOVE
			indirectForwardingPDR.FAR.State = smf_context.RULE_REMOVE
			pdrList = append(pdrList, indirectForwardingPDR)
			farList = append(farList, indirectForwardingPDR.FAR)
		}

		sendPFCPModification = true
		smContext.SetState(smf_context.PFCPModification)
		smContext.HoState = models.HoState_COMPLETED
		response.JsonData.HoState = models.HoState_COMPLETED
	}

	switch smContextUpdateData.Cause {
	case models.Cause_REL_DUE_TO_DUPLICATE_SESSION_ID:
		//* release PDU Session Here
		smContext.CheckState(smf_context.Active)
		// Wait till the state becomes Active again
		// TODO: implement sleep wait in concurrent architecture

		response.JsonData.N2SmInfo = &models.RefToBinaryData{ContentId: "PDUResourceReleaseCommand"}
		response.JsonData.N2SmInfoType = models.N2SmInfoType_PDU_RES_REL_CMD
		smContext.PDUSessionRelease_DUE_TO_DUP_PDU_ID = true

		if buf, err := smf_context.BuildPDUSessionResourceReleaseCommandTransfer(smContext); err != nil {
			smContext.Log.Errorf("Build PDUSessionResourceReleaseCommandTransfer failed: %v", err)
		} else {
			response.BinaryDataN2SmInformation = buf
		}

		smContext.Log.Infoln("[SMF] Cause_REL_DUE_TO_DUPLICATE_SESSION_ID")

		smContext.SetState(smf_context.PFCPModification)

		releaseTunnel(smContext)

		sendPFCPDelete = true
	}

	var httpResponse *http_wrapper.Response
	// Check FSM and take corresponding action
	switch smContext.State() {
	case smf_context.PFCPModification:
		smContext.Log.Traceln("In case PFCPModification")

		if sendPFCPModification {
			defaultPath := smContext.Tunnel.DataPathPool.GetDefaultPath()
			ANUPF := defaultPath.FirstDPNode
			pfcp_message.SendPfcpSessionModificationRequest(ANUPF.UPF.NodeID, smContext, pdrList, farList, barList, qerList)
		}

		if sendPFCPDelete {
			smContext.Log.Infoln("Send PFCP Deletion from HandlePDUSessionSMContextUpdate")
		}

		PFCPResponseStatus := <-smContext.SBIPFCPCommunicationChan

		switch PFCPResponseStatus {
		case smf_context.SessionUpdateSuccess:
			smContext.Log.Traceln("In case SessionUpdateSuccess")
			smContext.SetState(smf_context.Active)
			httpResponse = &http_wrapper.Response{
				Status: http.StatusOK,
				Body:   response,
			}
		case smf_context.SessionUpdateFailed:
			smContext.Log.Traceln("In case SessionUpdateFailed")
			smContext.SetState(smf_context.Active)
			// It is just a template
			httpResponse = &http_wrapper.Response{
				Status: http.StatusForbidden,
				Body: models.UpdateSmContextErrorResponse{
					JsonData: &models.SmContextUpdateError{
						Error: &Nsmf_PDUSession.N1SmError,
					},
				}, // Depends on the reason why N4 fail
			}

		case smf_context.SessionReleaseSuccess:
			smContext.Log.Traceln("In case SessionReleaseSuccess")
			smContext.SetState(smf_context.InActivePending)
			httpResponse = &http_wrapper.Response{
				Status: http.StatusOK,
				Body:   response,
			}

		case smf_context.SessionReleaseFailed:
			// Update SmContext Request(N1 PDU Session Release Request)
			// Send PDU Session Release Reject
			smContext.Log.Traceln("In case SessionReleaseFailed")
			problemDetail := models.ProblemDetails{
				Status: http.StatusInternalServerError,
				Cause:  "SYSTEM_FAILULE",
			}
			httpResponse = &http_wrapper.Response{
				Status: int(problemDetail.Status),
			}
			smContext.SetState(smf_context.Active)
			errResponse := models.UpdateSmContextErrorResponse{
				JsonData: &models.SmContextUpdateError{
					Error: &problemDetail,
				},
			}
			if buf, err := smf_context.BuildGSMPDUSessionReleaseReject(smContext); err != nil {
				smContext.Log.Errorf("build GSM PDUSessionReleaseReject failed: %+v", err)
			} else {
				errResponse.BinaryDataN1SmMessage = buf
			}

			errResponse.JsonData.N1SmMsg = &models.RefToBinaryData{ContentId: "PDUSessionReleaseReject"}
			httpResponse.Body = errResponse
		}
	case smf_context.ModificationPending:
		smContext.Log.Traceln("In case ModificationPending")
		smContext.SetState(smf_context.Active)
		httpResponse = &http_wrapper.Response{
			Status: http.StatusOK,
			Body:   response,
		}
	case smf_context.InActive, smf_context.InActivePending:
		smContext.Log.Traceln("In case InActive, InActivePending")
		httpResponse = &http_wrapper.Response{
			Status: http.StatusOK,
			Body:   response,
		}
	default:
		httpResponse = &http_wrapper.Response{
			Status: http.StatusOK,
			Body:   response,
		}
	}

	return httpResponse
}

func HandlePDUSessionSMContextRelease(smContextRef string, body models.ReleaseSmContextRequest) *http_wrapper.Response {
	logger.PduSessLog.Infoln("In HandlePDUSessionSMContextRelease")
	smContext := smf_context.GetSMContext(smContextRef)

	if smContext == nil {
		logger.PduSessLog.Warnf("SMContext[%s] is not found", smContextRef)

		httpResponse := &http_wrapper.Response{
			Header: nil,
			Status: http.StatusNotFound,
			Body: models.UpdateSmContextErrorResponse{
				JsonData: &models.SmContextUpdateError{
					UpCnxState: models.UpCnxState_DEACTIVATED,
					Error: &models.ProblemDetails{
						Type:   "Resource Not Found",
						Title:  "SMContext Ref is not found",
						Status: http.StatusNotFound,
					},
				},
			},
		}
		return httpResponse
	}

	smContext.SMLock.Lock()
	defer smContext.SMLock.Unlock()

	smContext.SetState(smf_context.PFCPModification)

	releaseTunnel(smContext)

	var httpResponse *http_wrapper.Response
	PFCPResponseStatus := <-smContext.SBIPFCPCommunicationChan

	switch PFCPResponseStatus {
	case smf_context.SessionReleaseSuccess:
		smContext.Log.Traceln("In case SessionReleaseSuccess")
		smContext.SetState(smf_context.InActivePending)
		httpResponse = &http_wrapper.Response{
			Status: http.StatusNoContent,
			Body:   nil,
		}

	case smf_context.SessionReleaseFailed:
		// Update SmContext Request(N1 PDU Session Release Request)
		// Send PDU Session Release Reject
		smContext.Log.Traceln("In case SessionReleaseFailed")
		problemDetail := models.ProblemDetails{
			Status: http.StatusInternalServerError,
			Cause:  "SYSTEM_FAILULE",
		}
		httpResponse = &http_wrapper.Response{
			Status: int(problemDetail.Status),
		}
		smContext.SetState(smf_context.Active)
		errResponse := models.UpdateSmContextErrorResponse{
			JsonData: &models.SmContextUpdateError{
				Error: &problemDetail,
			},
		}
		if buf, err := smf_context.BuildGSMPDUSessionReleaseReject(smContext); err != nil {
			smContext.Log.Errorf("Build GSM PDUSessionReleaseReject failed: %+v", err)
		} else {
			errResponse.BinaryDataN1SmMessage = buf
		}

		errResponse.JsonData.N1SmMsg = &models.RefToBinaryData{ContentId: "PDUSessionReleaseReject"}
		httpResponse.Body = errResponse
	default:
		smContext.Log.Warnf("The state shouldn't be [%s]\n", PFCPResponseStatus)

		smContext.Log.Traceln("In case Unknown")
		problemDetail := models.ProblemDetails{
			Status: http.StatusInternalServerError,
			Cause:  "SYSTEM_FAILULE",
		}
		httpResponse = &http_wrapper.Response{
			Status: int(problemDetail.Status),
		}
		smContext.SetState(smf_context.Active)
		errResponse := models.UpdateSmContextErrorResponse{
			JsonData: &models.SmContextUpdateError{
				Error: &problemDetail,
			},
		}
		if buf, err := smf_context.BuildGSMPDUSessionReleaseReject(smContext); err != nil {
			smContext.Log.Errorf("Build GSM PDUSessionReleaseReject failed: %+v", err)
		} else {
			errResponse.BinaryDataN1SmMessage = buf
		}

		errResponse.JsonData.N1SmMsg = &models.RefToBinaryData{ContentId: "PDUSessionReleaseReject"}
		httpResponse.Body = errResponse
	}

	smf_context.RemoveSMContext(smContext.Ref)

	return httpResponse
}

func releaseTunnel(smContext *smf_context.SMContext) {
	deletedPFCPNode := make(map[string]bool)
	smContext.PendingUPF = make(smf_context.PendingUPF)
	for _, dataPath := range smContext.Tunnel.DataPathPool {
		dataPath.DeactivateTunnelAndPDR(smContext)
		for curDataPathNode := dataPath.FirstDPNode; curDataPathNode != nil; curDataPathNode = curDataPathNode.Next() {
			curUPFID, err := curDataPathNode.GetUPFID()
			if err != nil {
				logger.PduSessLog.Error(err)
				continue
			}
			if _, exist := deletedPFCPNode[curUPFID]; !exist {
				pfcp_message.SendPfcpSessionDeletionRequest(curDataPathNode.UPF.NodeID, smContext)
				deletedPFCPNode[curUPFID] = true
				smContext.PendingUPF[curDataPathNode.GetNodeIP()] = true
			}
		}
	}
}
