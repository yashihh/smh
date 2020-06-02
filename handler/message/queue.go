package message

import (
	"free5gc/lib/http_wrapper"
	"free5gc/src/smf/logger"
	"sync"
)

var RspQueue *ResponseQueue

type ResponseQueue struct {
	RspQueue sync.Map
}

func init() {
	RspQueue = NewQueue()
}

func NewQueue() *ResponseQueue {
	var rq ResponseQueue
	return &rq
}

func (rq *ResponseQueue) PutItem(seqNum uint32, rspChan chan HandlerResponseMessage, response http_wrapper.Response) {

	Item := new(ResponseQueueItem)
	Item.RspChan = rspChan
	Item.Response = response
	rq.RspQueue.Store(seqNum, Item)
}

func (rq *ResponseQueue) GetItem(seqNum uint32) *ResponseQueueItem {
	item, exist := rq.RspQueue.Load(seqNum)
	if !exist {
		logger.CtxLog.Warnln("Http Response Queue has no ", seqNum, "'s item")
	}

	return item.(*ResponseQueueItem)
}

func (rq *ResponseQueue) DeleteItem(seqNum uint32) {
	rq.RspQueue.Delete(seqNum)
}

func (rq *ResponseQueue) CheckItemExist(seqNum uint32) (exist bool) {
	_, exist = rq.RspQueue.Load(seqNum)
	return exist
}
