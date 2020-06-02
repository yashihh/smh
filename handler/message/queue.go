package message

import (
	"free5gc/lib/http_wrapper"
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
	item, _ := rq.RspQueue.Load(seqNum)

	return item.(*ResponseQueueItem)
}

func (rq *ResponseQueue) DeleteItem(seqNum uint32) {
	rq.RspQueue.Delete(seqNum)
}

func (rq *ResponseQueue) CheckItemExist(seqNum uint32) (exist bool) {
	_, exist = rq.RspQueue.Load(seqNum)
	return exist
}
