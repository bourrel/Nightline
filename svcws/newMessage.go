package svcws

import (
	"context"
	"fmt"
	"svcdb"
	"time"
)

func sendMessageToUser(s Service, ctx context.Context, req newMessageRequest, user svcdb.User) {
	notif := &recvMessageNotif{
		UserName: user.Pseudo,
		UserID:   user.ID,
		Message:  req.Message.Text,
	}
	s.svcevent.Push(ctx, MessageRcvd, notif, req.Message.To)
}

func sendMessageToGroup(s Service, ctx context.Context, req newMessageRequest, group svcdb.Group, user svcdb.User) {
	notif := &recvMessageNotif{
		GroupName: group.Name,
		GroupID:   group.ID,
		UserName:  user.Pseudo,
		UserID:    user.ID,
		Message:   req.Message.Text,
	}

	for i := 0; i < len(group.Users); i++ {
		if group.Users[i].ID == req.Message.From {
			continue
		}

		s.svcevent.Push(ctx, GroupMessageRcvd, notif, group.Users[i].ID)
	}
	if group.Owner.ID != req.Message.From {
		s.svcevent.Push(ctx, GroupMessageRcvd, notif, group.Owner.ID)
	}
}

/*************** Service ***************/
func (s Service) NewMessage(ctx context.Context, v interface{}) (interface{}, error) {
	var req newMessageRequest

	mp, ok := v.(map[string]interface{})
	if !ok {
		fmt.Println("nok 1")
		return nil, nil
	}

	req.Message.From = int64(mp["from"].(float64))
	req.Message.To = int64(mp["to"].(float64))
	req.Message.Text = mp["message"].(string)

	fmt.Println(req)

	nType, err := s.svcdb.GetNodeType(ctx, req.Message.To)
	if err != nil {
		fmt.Println("NewMessage (GetNodeType) : " + err.Error())
		return nil, err
	}

	_, err = s.svcdb.CreateMessage(ctx, nType, req.Message)
	if err != nil {
		fmt.Println("NewMessage (CreateMessage)  : " + err.Error())
		return nil, err
	}

	user, err := s.svcdb.GetUserByID(ctx, req.Message.From)
	if err != nil {
		fmt.Println("NewMessage (GetUserByID)  : " + err.Error())
		return nil, err
	}

	if nType == "USER" {
		sendMessageToUser(s, ctx, req, user)
	} else if nType == "GROUP" {
		group, err := s.svcdb.GetGroup(ctx, req.Message.To)
		if err != nil {
			fmt.Println("NewMessage (GetGroup)  : " + err.Error())
			return nil, err
		}
		fmt.Println(group)

		sendMessageToGroup(s, ctx, req, group, user)
	}

	return newMessageResponse{}, nil
}

// /*************** Endpoint ***************/
type newMessageRequest struct {
	Message svcdb.Message `json:"message"`
}

type newMessageResponse struct {
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) NewMessage(ctx context.Context, v interface{}) (interface{}, error) {
	i, err := mw.next.NewMessage(ctx, v)

	mw.logger.Log(
		"method", "NewMessage",
		"took", time.Since(time.Now()),
	)

	return i, err
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) NewMessage(ctx context.Context, v interface{}) (interface{}, error) {
	return mw.next.NewMessage(ctx, v)
}
