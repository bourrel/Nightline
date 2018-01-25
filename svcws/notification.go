package svcws

import (
	"encoding/json"
	"fmt"

	"github.com/Shopify/sarama"
)

type recvMessageNotif struct {
	UserName  string `json:"userName"`
	UserID    int64  `json:"userID"`
	GroupName string `json:"groupName,omitempty"`
	GroupID   int64  `json:"groupID,omitempty"`
	Message   string `json:"message"`
}

type messageBody interface{}

type notificationMessage struct {
	Body messageBody `json:"body"`
	Name string      `json:"name"`
	User int64       `json:"user"`
}

func messageToResponse(nm *sarama.ConsumerMessage, currUser int64) interface{} {
	var ret notificationMessage

	err := json.Unmarshal(nm.Value, &ret)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	if ret.User != currUser {
		return nil
	}

	return ret
}

func SendMessageToConsumers(m chan *sarama.ConsumerMessage, message *sarama.ConsumerMessage) {
	for i := 0; i < clientCount; i++ {
		m <- message
	}
}
