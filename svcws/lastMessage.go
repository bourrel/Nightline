package svcws

import (
	"context"
	"fmt"
	"svcdb"
	"time"
)

/*************** Service ***************/
func (s Service) LastMessage(ctx context.Context, v interface{}) (lastMessageResponse, error) {
	var req lastMessageRequest

	mp, ok := v.(map[string]interface{})
	if !ok {
		fmt.Println("nok 1")
		return lastMessageResponse{}, nil
	}

	req.Initiator = int64(mp["initiator"].(float64))
	req.Recipient = int64(mp["recipient"].(float64))

	fmt.Println(req)

	messages, err := s.svcdb.GetLastMessages(ctx, req.Recipient, req.Initiator)
	if err != nil {
		fmt.Println("LastMessage  : " + err.Error())
		return lastMessageResponse{}, err
	}

	return lastMessageResponse{messages}, nil
}

// /*************** Endpoint ***************/
type lastMessageRequest struct {
	Initiator int64 `json:"initiator"`
	Recipient int64 `json:"recipient"`
	// messageCount int64 `json:"messageCount"`
}

type lastMessageResponse struct {
	Messages []svcdb.Message `json:"messages"`
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) LastMessage(ctx context.Context, v interface{}) (lastMessageResponse, error) {
	i, err := mw.next.LastMessage(ctx, v)

	mw.logger.Log(
		"method", "LastMessage",
		"took", time.Since(time.Now()),
	)

	return i, err
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) LastMessage(ctx context.Context, v interface{}) (lastMessageResponse, error) {
	return mw.next.LastMessage(ctx, v)
}
