package pdusession

import (
	"context"
	"free5gc/lib/CommonConsumerTestData/SMF/TestPDUSession"
	"free5gc/lib/openapi/Nsmf_PDUSession"
	"free5gc/lib/openapi/models"

	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPostSmContexts(t *testing.T) {

	go DummyServer()
	configuration := Nsmf_PDUSession.NewConfiguration()
	configuration.SetBasePath("https://127.0.0.10:29502")
	client := Nsmf_PDUSession.NewAPIClient(configuration)
	var request models.PostSmContextsRequest

	table := TestPDUSession.ConsumerSMFPDUSessionSMContextCreateTable["Service Request"]

	request.JsonData = &table

	request.BinaryDataN1SmMessage = TestPDUSession.GetEstablishmentRequestData(TestPDUSession.SERVICE_REQUEST)

	_, httpRsp, err := client.SMContextsCollectionApi.PostSmContexts(context.Background(), request)
	assert.True(t, err == nil, err)
	assert.True(t, httpRsp != nil)
	assert.Equal(t, "201 Created", httpRsp.Status)
}
