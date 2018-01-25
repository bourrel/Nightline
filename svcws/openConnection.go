package svcws

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Shopify/sarama"
	mux "github.com/gorilla/mux"
	ws "github.com/gorilla/websocket"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/log"
	httptransport "github.com/go-kit/kit/transport/http"
)

var upgrader = ws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type OpenConnectionID struct {
	UserID int64 `json:"id"`
}

func DecodeHTTPOpenConnection(_ context.Context, r *http.Request) (int64, error) {
	if err := r.ParseForm(); err != nil {
		fmt.Println("DecodeHTTPOpenConnection : " + err.Error())
		return -1, err
	}

	ID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		fmt.Println("DecodeHTTPOpenConnection : " + err.Error())
		return -1, err
	}

	return ID, nil
}

/*************** Service ***************/
func (s Service) OpenConnection(h http.Handler, logger log.Logger, message chan *sarama.ConsumerMessage) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var response chan interface{}

			ctx := r.Context()
			usr, err := DecodeHTTPOpenConnection(ctx, r)
			if err != nil {
				fmt.Println("OpenConnection failure : " + err.Error())
				return
			}
			currUser := usr
			clientCount++

			// Upgrade connection to WS
			c, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				fmt.Println("OpenConnection failure : " + err.Error())
				return
			}
			logger.Log("New client", usr)

			// Close socket
			defer func() {
				logger.Log("Closing socket...", usr)

				clientCount--
				response <- "close"
				c.Close()
			}()

			response = make(chan interface{})
			go WriteSocket(c, response, message, currUser)
			for {
				var ocRequest OpenConnectionRequest
				err := c.ReadJSON(&ocRequest)
				if err != nil {
					fmt.Println("OpenConnection Error when reading", err)
					break
				}

				err = readRequest(s, ctx, ocRequest, response)
				if err != nil {
					fmt.Println("OpenConnection Error when writing", err)
					break
				}
			}
		})
	}(h)
}

/*************** Transport ***************/
func OpenConnectionHTTPHandler(tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router,
	svc IService, options []httptransport.ServerOption, c chan *sarama.ConsumerMessage) *mux.Route {
	route :=
		r.Methods("GET").
			Path("/{id}").
			Handler(func(svc IService, r *mux.Router, c chan *sarama.ConsumerMessage) http.Handler {
				var h http.Handler

				return svc.OpenConnection(h, logger, c)
			}(svc, r, c))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) OpenConnection(h http.Handler, logger log.Logger, message chan *sarama.ConsumerMessage) http.Handler {
	err := mw.next.OpenConnection(h, logger, message)

	mw.logger.Log(
		"method", "OpenConnection",
		"took", time.Since(time.Now()),
	)

	return err
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) OpenConnection(h http.Handler, logger log.Logger, message chan *sarama.ConsumerMessage) http.Handler {
	return mw.next.OpenConnection(h, logger, message)
}
