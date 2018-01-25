package svcws

import (
	"context"
	"fmt"

	"github.com/Shopify/sarama"
	ws "github.com/gorilla/websocket"
)

type wsBody interface{}

type OpenConnectionRequest struct {
	Name string `json:"name"`
	Body wsBody `json:"body"`
}

type OpenConnectionResponse struct {
	Name string `json:"name"`
	Body wsBody `json:"body"`
}

func WriteSocket(c *ws.Conn, response chan interface{}, message chan *sarama.ConsumerMessage, currUser int64) {
	for {
		select {
		case consumerMsg := <-message:
			if msg := messageToResponse(consumerMsg, currUser); msg != nil {
				fmt.Println("user", currUser, "received a message.")
				err := c.WriteJSON(msg)
				if err != nil {
					fmt.Println("WriteSocket Consumer error when writing", err)
					break
				}
			}
		case responseMsg := <-response:
			if str, ok := responseMsg.(string); ok && str == "close" {
				c.Close()
				close(response)
				return
			}

			fmt.Println(responseMsg)
			err := c.WriteJSON(responseMsg)
			if err != nil {
				fmt.Println("WriteSocket Message error when writing", err)
				break
			}
		}
	}
}

func readRequest(svc Service, ctx context.Context, ocRequest OpenConnectionRequest, response chan interface{}) error {
	var tmpResponse interface{}
	var ocResponse OpenConnectionResponse
	var err error

	switch ocRequest.Name {
	case "get_last_messages":
		tmpResponse, err = svc.LastMessage(ctx, ocRequest.Body)
		ocResponse.Name = "last_messages"
		ocResponse.Body = tmpResponse
		break
	case "new_message":
		_, err = svc.NewMessage(ctx, ocRequest.Body)
		break
	default:
		fmt.Println("Error invalid request")
	}

	// Send response to websocket
	if tmpResponse != nil && tmpResponse != (OpenConnectionResponse{}) {
		response <- tmpResponse
	}
	return err
}
